// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package stubs

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/libs/num"
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

func (ts *TopologyStub) RecalcValidatorSet(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) []*types.PartyContributionScore {
	return []*types.PartyContributionScore{}
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
