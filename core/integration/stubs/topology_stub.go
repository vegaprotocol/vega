// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package stubs

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"code.vegaprotocol.io/vega/core/validators"
)

type TopologyStub struct {
	validators    map[string]string
	nodeID        string
	broker        *BrokerStub
	minDelegation num.Decimal
}

func NewTopologyStub(nodeID string, broker *BrokerStub) *TopologyStub {
	return &TopologyStub{
		validators: map[string]string{},
		nodeID:     nodeID,
		broker:     broker,
	}
}

func (ts *TopologyStub) OnMinDelegationUpdated(_ context.Context, minDelegation num.Decimal) error {
	ts.minDelegation = minDelegation
	return nil
}

func (ts *TopologyStub) Len() int {
	return len(ts.validators)
}

func (ts *TopologyStub) ValidatorPerformanceScore(nodeID string) num.Decimal {
	return num.DecimalFromFloat(1)
}

func (ts *TopologyStub) SelfNodeID() string {
	return ts.nodeID
}

func (ts *TopologyStub) SelfVegaPubKey() string {
	return ts.nodeID
}

func (ts *TopologyStub) IsValidator() bool {
	return true
}

func (ts *TopologyStub) IsValidatorVegaPubKey(pubKey string) bool {
	return true
}

func (ts *TopologyStub) IsTendermintValidator(pubKey string) bool {
	return true
}

func (ts *TopologyStub) IsValidatorNodeID(nodeID string) bool {
	_, ok := ts.validators[nodeID]
	return ok
}

func (ts *TopologyStub) RecalcValidatorSet(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) []*types.PartyContibutionScore {
	return []*types.PartyContibutionScore{}
}

func (ts *TopologyStub) AllNodeIDs() []string {
	nodes := make([]string, 0, len(ts.validators))
	for n := range ts.validators {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	return nodes
}

func (ts *TopologyStub) AllVegaPubKeys() []string {
	nodes := make([]string, 0, len(ts.validators))
	for _, pk := range ts.validators {
		nodes = append(nodes, pk)
	}
	sort.Strings(nodes)
	return nodes
}

func (ts *TopologyStub) Get(key string) *validators.ValidatorData {
	if data, ok := ts.validators[key]; ok {
		return &validators.ValidatorData{
			ID:         key,
			VegaPubKey: data,
		}
	}

	return nil
}

func (ts *TopologyStub) AddValidator(node string, pubkey string) {
	ts.validators[node] = pubkey
}

func (ts *TopologyStub) GetRewardsScores(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) (*types.ScoreData, *types.ScoreData) {
	tmScores := ts.calculateTMScores(delegationState, stakeScoreParams)
	evts := make([]events.Event, 0, len(tmScores.NodeIDSlice))
	for _, nodeID := range tmScores.NodeIDSlice {
		evts = append(evts, events.NewValidatorScore(ctx, nodeID, epochSeq, tmScores.ValScores[nodeID], tmScores.NormalisedScores[nodeID], tmScores.RawValScores[nodeID], tmScores.PerformanceScores[nodeID], tmScores.MultisigScores[nodeID], "tendermint"))
	}

	ts.broker.SendBatch(evts)

	return tmScores, &types.ScoreData{}
}

// calculateTMScores returns the reward validator scores for the tendermint validatore.
func (ts *TopologyStub) calculateTMScores(delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) *types.ScoreData {
	tmScores := &types.ScoreData{}
	validatorSet := make(map[string]struct{}, len(delegationState))
	tmScores.PerformanceScores = make(map[string]num.Decimal, len(delegationState))
	for _, ds := range delegationState {
		validatorSet[ds.NodeID] = struct{}{}
		if ds.SelfStake.ToDecimal().GreaterThanOrEqual(ts.minDelegation) {
			tmScores.PerformanceScores[ds.NodeID] = num.DecimalFromFloat(1)
		} else {
			tmScores.PerformanceScores[ds.NodeID] = num.DecimalZero()
		}
	}

	tmDelegation, tmTotalDelegation := validators.CalcDelegation(validatorSet, delegationState)
	optStake := validators.GetOptimalStake(tmTotalDelegation, len(tmDelegation), stakeScoreParams)
	tv := validators.CalcAntiWhalingScore(tmDelegation, tmTotalDelegation, optStake, stakeScoreParams)

	tmScores.RawValScores = tv
	tmScores.ValScores = make(map[string]num.Decimal, len(tv))
	for k, d := range tv {
		tmScores.ValScores[k] = d.Mul(tmScores.PerformanceScores[k])
	}

	// normalise the scores
	tmScores.NormalisedScores = ts.normaliseScores(tmScores.ValScores)

	// sort the list of tm validators
	tmNodeIDs := make([]string, 0, len(tv))
	for k := range tv {
		tmNodeIDs = append(tmNodeIDs, k)
	}

	sort.Strings(tmNodeIDs)
	tmScores.NodeIDSlice = tmNodeIDs
	return tmScores
}

func (ts *TopologyStub) normaliseScores(scores map[string]num.Decimal) map[string]num.Decimal {
	totalScore := num.DecimalZero()
	for _, v := range scores {
		totalScore = totalScore.Add(v)
	}

	normScores := make(map[string]num.Decimal, len(scores))
	for n, s := range scores {
		if totalScore.IsPositive() {
			normScores[n] = s.Div(totalScore)
		} else {
			normScores[n] = num.DecimalZero()
		}
	}
	return normScores
}

func (*TopologyStub) GetVotingPower(pubkey string) int64 {
	return 1
}

func (*TopologyStub) GetTotalVotingPower() int64 {
	return 1
}
