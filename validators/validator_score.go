package validators

import (
	"context"
	"math/rand"
	"sort"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type valScore struct {
	ID    string
	score num.Decimal
}

// getStakeScore returns a score for the validator based on their relative score of the total score.
// No anti-whaling is applied.
func getStakeScore(delegationState []*types.ValidatorData, minimumStake *num.Uint) map[string]num.Decimal {
	totalStake := num.Zero()
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
			scores[ds.NodeID] = t.validatorPerformance.ValidatorPerformanceScore(vd.data.TmPubKey, vd.validatorPower, totalTmPower)
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
	stakeScores := getStakeScore(delegationState, t.minimumStake)
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
func CalcValidatorScore(valStake, totalStake, numVal num.Decimal, stakeScoreParams types.StakeScoreParams) num.Decimal {
	if totalStake.IsZero() {
		return num.DecimalZero()
	}
	a := num.MaxD(stakeScoreParams.MinVal, numVal.Div(stakeScoreParams.CompLevel))
	optStake := totalStake.Div(a)
	penaltyFlatAmt := num.MaxD(num.DecimalZero(), valStake.Sub(optStake))
	penaltyDownAmt := num.MaxD(num.DecimalZero(), valStake.Sub(stakeScoreParams.OptimalStakeMultiplier.Mul(optStake)))
	linearScore := valStake.Sub(penaltyFlatAmt).Sub(penaltyDownAmt).Div(totalStake) // totalStake guaranteed to be non zero at this point
	decimal1, _ := num.DecimalFromString("1.0")
	linearScore = num.MinD(decimal1, num.MaxD(num.DecimalZero(), linearScore))
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
func getMultisigScore(log *logging.Logger, rawScores map[string]num.Decimal, perfScore map[string]num.Decimal, multiSigTopology MultiSigTopology, numberEthMultisigSigners int, nodeIDToEthAddress map[string]string) map[string]num.Decimal {
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
		if valScores[i].score == valScores[j].score {
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
		}
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
	tmScores := t.calculateTMScores(delegationState, stakeScoreParams)
	ezScores := t.calculateErsatzScores(delegationState)

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

// calculateErsatzScores returns the reward validator scores for the ersatz validatore.
func (t *Topology) calculateErsatzScores(delegationState []*types.ValidatorData) *types.ScoreData {
	ezScores := &types.ScoreData{}
	ev := map[string]num.Decimal{}
	evStake := map[string]*num.Uint{}
	evTotal := num.Zero()
	evDelegation := []*types.ValidatorData{}

	// split the delegation into tendermint and ersatz and count their respective totals
	for _, ds := range delegationState {
		if t.validators[ds.NodeID].status == ValidatorStatusErsatz {
			ev[ds.NodeID] = num.DecimalZero()
			stake := num.Sum(ds.SelfStake, ds.StakeByDelegators)
			evStake[ds.NodeID] = stake
			evTotal.AddSum(stake)
			evDelegation = append(evDelegation, ds)
		}
	}
	evTotalD := evTotal.ToDecimal()

	// calculate a simple stake score (no anti-whaling)
	for k, v := range evStake {
		if evTotalD.IsPositive() {
			ev[k] = v.ToDecimal().Div(evTotalD)
		} else {
			ev[k] = num.DecimalZero()
		}
	}

	ezScores.RawValScores = ev

	// calculate the performance score based on the number of times they signed vs the expected number of times in the last 10 rounds
	ezScores.PerformanceScores = t.getPerformanceScore(evDelegation)

	ezScores.MultisigScores = make(map[string]num.Decimal, len(ev))
	for k := range ev {
		ezScores.MultisigScores[k] = decimalOne
	}

	// calculate the validator score as raw_score x performance_score
	ezScores.ValScores = getValScore(ezScores.RawValScores, ezScores.PerformanceScores)

	// normalise the scores
	ezScores.NormalisedScores = normaliseScores(ezScores.ValScores, t.rng)

	ezNodeIDs := make([]string, 0, len(ev))
	for k := range ev {
		ezNodeIDs = append(ezNodeIDs, k)
	}
	sort.Strings(ezNodeIDs)
	ezScores.NodeIDSlice = ezNodeIDs

	return ezScores
}

// CalcDelegation extracts the delegation of the validator set from the delegation state slice and returns the total delegation.
func CalcDelegation(validators map[string]struct{}, delegationState []*types.ValidatorData) ([]*types.ValidatorData, num.Decimal) {
	tv := map[string]num.Decimal{}
	tvTotal := num.Zero()
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

// CalcAntiWhalingScore calculates the anti-whaling stake score for the validators represented in the given delegation set.
func CalcAntiWhalingScore(delegationState []*types.ValidatorData, totalStakeD num.Decimal, stakeScoreParams types.StakeScoreParams) map[string]num.Decimal {
	stakeScore := make(map[string]num.Decimal, len(delegationState))
	for _, ds := range delegationState {
		if totalStakeD.IsPositive() {
			stakeScore[ds.NodeID] = CalcValidatorScore(num.Sum(ds.SelfStake, ds.StakeByDelegators).ToDecimal(), totalStakeD, num.DecimalFromInt64(int64(len(delegationState))), stakeScoreParams)
		} else {
			stakeScore[ds.NodeID] = num.DecimalZero()
		}
	}
	return stakeScore
}

// calculateTMScores returns the reward validator scores for the tendermint validators.
func (t *Topology) calculateTMScores(delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) *types.ScoreData {
	tmScores := &types.ScoreData{}

	// identify tendermint validators for epoch
	tmValidators := map[string]struct{}{}
	nodeIDToEthAddress := map[string]string{}
	for k, d := range t.validators {
		if d.status == ValidatorStatusTendermint {
			tmValidators[k] = struct{}{}
		}
		nodeIDToEthAddress[d.data.ID] = d.data.EthereumAddress
	}

	// calculate the delegation and anti-whaling score for the tendermint validators
	tmDelegation, tmTotalDelegation := CalcDelegation(tmValidators, delegationState)
	tmStakeScore := CalcAntiWhalingScore(tmDelegation, tmTotalDelegation, stakeScoreParams)

	tmScores.RawValScores = tmStakeScore

	// calculate performance score based on tm performance
	tmScores.PerformanceScores = t.getPerformanceScore(tmDelegation)

	// calculate multisig score for the tm validators
	tmScores.MultisigScores = getMultisigScore(t.log, tmScores.RawValScores, tmScores.PerformanceScores, t.multiSigTopology, t.numberEthMultisigSigners, nodeIDToEthAddress)

	// calculate the final score
	tmScores.ValScores = getValScore(tmScores.RawValScores, tmScores.PerformanceScores, tmScores.MultisigScores)

	// normalise the scores
	tmScores.NormalisedScores = normaliseScores(tmScores.ValScores, t.rng)

	// sort the list of tm validators
	tmNodeIDs := make([]string, 0, len(tmDelegation))
	for k := range tmStakeScore {
		tmNodeIDs = append(tmNodeIDs, k)
	}

	sort.Strings(tmNodeIDs)
	tmScores.NodeIDSlice = tmNodeIDs

	for _, k := range tmNodeIDs {
		t.log.Info("reward scores for", logging.String("node-id", k), logging.String("stake-score", tmScores.RawValScores[k].String()), logging.String("performance-score", tmScores.PerformanceScores[k].String()), logging.String("multisig-score", tmScores.MultisigScores[k].String()), logging.String("validator-score", tmScores.ValScores[k].String()), logging.String("normalised-score", tmScores.NormalisedScores[k].String()))
	}

	return tmScores
}
