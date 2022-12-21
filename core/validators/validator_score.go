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

package validators

import (
	"context"
	"math/rand"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

type valScore struct {
	ID    string
	score num.Decimal
}

// getStakeScore returns a score for the validator based on their relative score of the total score.
// No anti-whaling is applied.
func getStakeScore(delegationState []*types.ValidatorData) map[string]num.Decimal {
	totalStake := num.UintZero()
	for _, ds := range delegationState {
		totalStake.AddSum(num.Sum(ds.SelfStake, ds.StakeByDelegators))
	}

	totalStakeD := totalStake.ToDecimal()
	scores := make(map[string]num.Decimal, len(delegationState))
	for _, ds := range delegationState {
		if totalStakeD.IsPositive() {
			scores[ds.NodeID] = num.Sum(ds.SelfStake, ds.StakeByDelegators).ToDecimal().Div(totalStakeD)
		} else {
			scores[ds.NodeID] = num.DecimalZero()
		}
	}
	return scores
}

// getPerformanceScore returns the performance score of the validators.
// if the node has been a tendermint validator for the epoch it returns its tendermint performance score (as the ratio between the blocks proposed
// and the number of times it was expected to propose)
// if the node has less than the minimum stake they get 0 performance score
// if the node is ersatz or waiting list validators and has not yet forwarded or voted first - their score is 0
// if the node is not tm node - their score is based on the number of times out of the last 10 that they signed every 1000 blocks.
func (t *Topology) getPerformanceScore(delegationState []*types.ValidatorData) map[string]num.Decimal {
	scores := make(map[string]num.Decimal, len(delegationState))

	totalTmPower := int64(0)
	for _, vs := range t.validators {
		totalTmPower += vs.validatorPower
	}

	for _, ds := range delegationState {
		vd := t.validators[ds.NodeID]
		performanceScore := num.DecimalZero()
		if ds.SelfStake.LT(t.minimumStake) {
			scores[ds.NodeID] = performanceScore
			continue
		}
		if vd.status == ValidatorStatusTendermint {
			scores[ds.NodeID] = t.validatorPerformance.ValidatorPerformanceScore(vd.data.TmPubKey, vd.validatorPower, totalTmPower, t.performanceScalingFactor)
			continue
		}

		if vd.numberOfEthereumEventsForwarded < t.minimumEthereumEventsForNewValidator {
			scores[ds.NodeID] = performanceScore
			continue
		}
		for _, v := range t.validators[ds.NodeID].heartbeatTracker.blockSigs {
			if v {
				performanceScore = performanceScore.Add(PerformanceIncrement)
			}
		}
		scores[ds.NodeID] = performanceScore
	}

	return scores
}

// getRankingScore returns the score for ranking as stake_score x performance_score.
// for validators in ersatz or tendermint it is scaled by 1+incumbent factor.
func (t *Topology) getRankingScore(delegationState []*types.ValidatorData) (map[string]num.Decimal, map[string]num.Decimal, map[string]num.Decimal) {
	stakeScores := getStakeScore(delegationState)
	performanceScores := t.getPerformanceScore(delegationState)
	rankingScores := t.getRankingScoreInternal(stakeScores, performanceScores)
	return stakeScores, performanceScores, rankingScores
}

func (t *Topology) getRankingScoreInternal(stakeScores, perfScores map[string]num.Decimal) map[string]num.Decimal {
	if len(stakeScores) != len(perfScores) {
		t.log.Panic("incompatible slice length for stakeScores and perfScores")
	}
	rankingScore := make(map[string]num.Decimal, len(stakeScores))
	for nodeID := range stakeScores {
		vd := t.validators[nodeID]
		ranking := stakeScores[nodeID].Mul(perfScores[nodeID])
		if vd.status == ValidatorStatusTendermint || vd.status == ValidatorStatusErsatz {
			ranking = ranking.Mul(t.validatorIncumbentBonusFactor)
		}
		rankingScore[nodeID] = ranking
	}
	return rankingScore
}

// normaliseScores normalises the given scores with respect to their sum, making sure they don't go above 1.
func normaliseScores(scores map[string]num.Decimal, rng *rand.Rand) map[string]num.Decimal {
	totalScore := num.DecimalZero()
	for _, v := range scores {
		totalScore = totalScore.Add(v)
	}

	normScores := make(map[string]num.Decimal, len(scores))
	if totalScore.IsZero() {
		for k := range scores {
			normScores[k] = num.DecimalZero()
		}
		return normScores
	}

	scoreSum := num.DecimalZero()
	for n, s := range scores {
		normScores[n] = s.Div(totalScore)
		scoreSum = scoreSum.Add(normScores[n])
	}
	if scoreSum.LessThanOrEqual(DecimalOne) {
		return normScores
	}
	keys := make([]string, 0, len(normScores))
	for k := range normScores {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	precisionError := scoreSum.Sub(num.DecimalFromInt64(1))
	unluckyValidator := rng.Intn(len(keys))
	normScores[keys[unluckyValidator]] = num.MaxD(normScores[keys[unluckyValidator]].Sub(precisionError), num.DecimalZero())
	return normScores
}

// calcValidatorScore calculates the stake based raw validator score with anti whaling.
func CalcValidatorScore(valStake, totalStake, optStake num.Decimal, stakeScoreParams types.StakeScoreParams) num.Decimal {
	if totalStake.IsZero() {
		return num.DecimalZero()
	}
	return antiwhale(valStake, totalStake, optStake, stakeScoreParams)
}

func antiwhale(valStake, totalStake, optStake num.Decimal, stakeScoreParams types.StakeScoreParams) num.Decimal {
	penaltyFlatAmt := num.MaxD(num.DecimalZero(), valStake.Sub(optStake))
	penaltyDownAmt := num.MaxD(num.DecimalZero(), valStake.Sub(stakeScoreParams.OptimalStakeMultiplier.Mul(optStake)))
	linearScore := valStake.Sub(penaltyFlatAmt).Sub(penaltyDownAmt).Div(totalStake) // totalStake guaranteed to be non zero at this point
	linearScore = num.MinD(num.DecimalOne(), num.MaxD(num.DecimalZero(), linearScore))
	return linearScore
}

// getValScore returns the multiplications of the corresponding score for each validator.
func getValScore(inScores ...map[string]num.Decimal) map[string]num.Decimal {
	if len(inScores) == 0 {
		return map[string]num.Decimal{}
	}
	scores := make(map[string]num.Decimal, len(inScores[0]))
	for k := range inScores[0] {
		s := num.DecimalFromFloat(1)
		for _, v := range inScores {
			s = s.Mul(v[k])
		}
		scores[k] = s
	}
	return scores
}

// getMultisigScore (applies to tm validators only) returns multisigScore as:
// if the val_score = raw_score x performance_score  is in the top <numberEthMultisigSigners> and the validator is on the multisig contract => 1
// else 0
// that means a validator in tendermint set only gets a reward if it is in the top <numberEthMultisigSigners> and their registered with the multisig contract.
func getMultisigScore(log *logging.Logger, status ValidatorStatus, rawScores map[string]num.Decimal, perfScore map[string]num.Decimal, multiSigTopology MultiSigTopology, numberEthMultisigSigners int, nodeIDToEthAddress map[string]string) map[string]num.Decimal {
	if status == ValidatorStatusErsatz {
		scores := make(map[string]num.Decimal, len(rawScores))
		for k := range rawScores {
			scores[k] = decimalOne
		}
		return scores
	}

	ethAddresses := make([]string, 0, len(rawScores))
	for k := range rawScores {
		if eth, ok := nodeIDToEthAddress[k]; !ok {
			log.Panic("missing eth address in mapping", logging.String("node-id", k))
		} else {
			ethAddresses = append(ethAddresses, eth)
		}
	}
	sort.Strings(ethAddresses)

	if multiSigTopology.ExcessSigners(ethAddresses) {
		res := make(map[string]num.Decimal, len(rawScores))
		for rs := range rawScores {
			res[rs] = num.DecimalZero()
		}
		return res
	}

	valScores := make([]valScore, 0, len(rawScores))
	for k, d := range rawScores {
		valScores = append(valScores, valScore{ID: k, score: d.Mul(perfScore[k])})
	}

	sort.SliceStable(valScores, func(i, j int) bool {
		if valScores[i].score.Equal(valScores[j].score) {
			return valScores[i].ID < valScores[j].ID
		}
		return valScores[i].score.GreaterThan(valScores[j].score)
	})

	res := make(map[string]num.Decimal, len(valScores))
	for i, vs := range valScores {
		if i < numberEthMultisigSigners {
			if eth, ok := nodeIDToEthAddress[vs.ID]; !ok {
				log.Panic("missing eth address in mapping", logging.String("node-id", vs.ID))
			} else {
				if multiSigTopology.IsSigner(eth) {
					res[vs.ID] = decimalOne
				}
			}
			continue
		}
		// everyone else is a 1
		res[vs.ID] = decimalOne
	}
	for k := range rawScores {
		if _, ok := res[k]; !ok {
			res[k] = num.DecimalZero()
		}
	}

	return res
}

// GetRewardsScores returns the reward scores (raw, performance, multisig, validator_score, and normalised) for tm and ersatz validaor sets.
func (t *Topology) GetRewardsScores(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) (*types.ScoreData, *types.ScoreData) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	tmScores, optStake := t.calculateScores(delegationState, ValidatorStatusTendermint, stakeScoreParams, nil)
	ezScores, _ := t.calculateScores(delegationState, ValidatorStatusErsatz, stakeScoreParams, &optStake)

	evts := make([]events.Event, 0, len(tmScores.NodeIDSlice)+len(ezScores.NodeIDSlice))
	for _, nodeID := range tmScores.NodeIDSlice {
		evts = append(evts, events.NewValidatorScore(ctx, nodeID, epochSeq, tmScores.ValScores[nodeID], tmScores.NormalisedScores[nodeID], tmScores.RawValScores[nodeID], tmScores.PerformanceScores[nodeID], tmScores.MultisigScores[nodeID], "tendermint"))
	}
	for _, nodeID := range ezScores.NodeIDSlice {
		evts = append(evts, events.NewValidatorScore(ctx, nodeID, epochSeq, ezScores.ValScores[nodeID], ezScores.NormalisedScores[nodeID], ezScores.RawValScores[nodeID], ezScores.PerformanceScores[nodeID], decimalOne, "ersatz"))
	}
	t.broker.SendBatch(evts)
	return tmScores, ezScores
}

func (t *Topology) calculateScores(delegationState []*types.ValidatorData, validatorStatus ValidatorStatus, stakeScoreParams types.StakeScoreParams, optStake *num.Decimal) (*types.ScoreData, num.Decimal) {
	scores := &types.ScoreData{}

	// identify validators for the status for the epoch
	validatorsForStatus := map[string]struct{}{}
	nodeIDToEthAddress := map[string]string{}
	for k, d := range t.validators {
		if d.status == validatorStatus {
			validatorsForStatus[k] = struct{}{}
		}
		nodeIDToEthAddress[d.data.ID] = d.data.EthereumAddress
	}

	// calculate the delegation and anti-whaling score for the validators with the given status
	delegationForStatus, totalDelegationForStatus := CalcDelegation(validatorsForStatus, delegationState)
	if optStake == nil {
		optimalkStake := GetOptimalStake(totalDelegationForStatus, len(delegationForStatus), stakeScoreParams)
		optStake = &optimalkStake
	}

	scores.RawValScores = CalcAntiWhalingScore(delegationForStatus, totalDelegationForStatus, *optStake, stakeScoreParams)

	// calculate performance score based on performance of the validators with the given status
	scores.PerformanceScores = t.getPerformanceScore(delegationForStatus)

	// calculate multisig score for the validators
	scores.MultisigScores = getMultisigScore(t.log, validatorStatus, scores.RawValScores, scores.PerformanceScores, t.multiSigTopology, t.numberEthMultisigSigners, nodeIDToEthAddress)

	// calculate the final score
	scores.ValScores = getValScore(scores.RawValScores, scores.PerformanceScores, scores.MultisigScores)

	// normalise the scores
	scores.NormalisedScores = normaliseScores(scores.ValScores, t.rng)

	// sort the list of tm validators
	tmNodeIDs := make([]string, 0, len(delegationForStatus))
	for k := range scores.RawValScores {
		tmNodeIDs = append(tmNodeIDs, k)
	}

	sort.Strings(tmNodeIDs)
	scores.NodeIDSlice = tmNodeIDs

	for _, k := range tmNodeIDs {
		t.log.Info("reward scores for", logging.String("node-id", k), logging.String("stake-score", scores.RawValScores[k].String()), logging.String("performance-score", scores.PerformanceScores[k].String()), logging.String("multisig-score", scores.MultisigScores[k].String()), logging.String("validator-score", scores.ValScores[k].String()), logging.String("normalised-score", scores.NormalisedScores[k].String()))
	}

	return scores, *optStake
}

// CalcDelegation extracts the delegation of the validator set from the delegation state slice and returns the total delegation.
func CalcDelegation(validators map[string]struct{}, delegationState []*types.ValidatorData) ([]*types.ValidatorData, num.Decimal) {
	tv := map[string]num.Decimal{}
	tvTotal := num.UintZero()
	tvDelegation := []*types.ValidatorData{}

	// split the delegation into tendermint and ersatz and count their respective totals
	for _, ds := range delegationState {
		if _, ok := validators[ds.NodeID]; ok {
			tv[ds.NodeID] = num.DecimalZero()
			stake := num.Sum(ds.SelfStake, ds.StakeByDelegators)
			tvTotal.AddSum(stake)
			tvDelegation = append(tvDelegation, ds)
		}
	}
	tvTotalD := tvTotal.ToDecimal()
	return tvDelegation, tvTotalD
}

func GetOptimalStake(tmTotalDelegation num.Decimal, numValidators int, params types.StakeScoreParams) num.Decimal {
	if tmTotalDelegation.IsPositive() {
		numVal := num.DecimalFromInt64(int64(numValidators))
		return tmTotalDelegation.Div(num.MaxD(params.MinVal, numVal.Div(params.CompLevel)))
	}
	return num.DecimalZero()
}

// CalcAntiWhalingScore calculates the anti-whaling stake score for the validators represented in the given delegation set.
func CalcAntiWhalingScore(delegationState []*types.ValidatorData, totalStakeD, optStake num.Decimal, stakeScoreParams types.StakeScoreParams) map[string]num.Decimal {
	stakeScore := make(map[string]num.Decimal, len(delegationState))
	for _, ds := range delegationState {
		if totalStakeD.IsPositive() {
			stakeScore[ds.NodeID] = CalcValidatorScore(num.Sum(ds.SelfStake, ds.StakeByDelegators).ToDecimal(), totalStakeD, optStake, stakeScoreParams)
		} else {
			stakeScore[ds.NodeID] = num.DecimalZero()
		}
	}
	return stakeScore
}
