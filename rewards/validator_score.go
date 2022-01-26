package rewards

import (
	"context"
	"math/rand"
	"sort"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type scoreData struct {
	rawValScores      map[string]num.Decimal
	performanceScores map[string]num.Decimal
	valScores         map[string]num.Decimal
	normalisedScores  map[string]num.Decimal
	nodeIDSlice       []string
}

func calcNormalisedScore(epochSeq string, validatorsData []*types.ValidatorData, minVal, compLevel num.Decimal, optimalStakeMultiplier num.Decimal, rng *rand.Rand, validatorPerformance ValidatorPerformance) *scoreData {
	// calculate the total amount of tokens delegated across all validators
	totalStake := calcTotalStake(validatorsData)
	totalScore := num.DecimalZero()
	rawScores := make(map[string]num.Decimal, len(validatorsData))
	performanceScores := make(map[string]num.Decimal, len(validatorsData))
	valScores := make(map[string]num.Decimal, len(validatorsData))
	normalisedScores := make(map[string]num.Decimal, len(validatorsData))
	if totalStake.IsZero() {
		return &scoreData{
			rawScores, performanceScores, valScores, normalisedScores, []string{},
		}
	}

	// for each validator calculate the score
	nodeIDSlice := []string{}
	for _, vd := range validatorsData {
		valStake := num.Sum(vd.StakeByDelegators, vd.SelfStake)
		rawValScore := calcValidatorScore(valStake.ToDecimal(), totalStake.ToDecimal(), minVal, compLevel, num.DecimalFromInt64(int64(len(validatorsData))), optimalStakeMultiplier)
		rawScores[vd.NodeID] = rawValScore
		perfScore := validatorPerformance.ValidatorPerformanceScore(vd.NodeID)
		performanceScores[vd.NodeID] = perfScore
		valScore := perfScore.Mul(rawValScore)
		valScores[vd.NodeID] = valScore
		totalScore = totalScore.Add(valScore)
		nodeIDSlice = append(nodeIDSlice, vd.NodeID)
	}

	sort.Strings(nodeIDSlice)
	scoreSum := num.DecimalZero()
	for _, k := range nodeIDSlice {
		score := valScores[k]
		if !totalScore.IsZero() {
			normalisedScores[k] = score.Div(totalScore)
		} else {
			normalisedScores[k] = num.DecimalZero()
		}

		scoreSum = scoreSum.Add(normalisedScores[k])
	}

	// verify that the sum of scores is 1, if not adjust one score at random
	if scoreSum.GreaterThan(num.DecimalFromInt64(1)) {
		precisionError := scoreSum.Sub(num.DecimalFromInt64(1))
		unluckyValidator := rng.Intn(len(nodeIDSlice))
		normalisedScores[nodeIDSlice[unluckyValidator]] = num.MaxD(normalisedScores[nodeIDSlice[unluckyValidator]].Sub(precisionError), num.DecimalZero())
	}

	return &scoreData{
		rawValScores:      rawScores,
		performanceScores: performanceScores,
		valScores:         valScores,
		normalisedScores:  normalisedScores,
		nodeIDSlice:       nodeIDSlice,
	}
}

// calculate the score for each validator and normalise by the total score.
func calcValidatorsNormalisedScore(ctx context.Context, broker Broker, epochSeq string, validatorsData []*types.ValidatorData, minVal, compLevel num.Decimal, optimalStakeMultiplier num.Decimal, rng *rand.Rand, validatorPerformance ValidatorPerformance) map[string]num.Decimal {
	scoreData := calcNormalisedScore(epochSeq, validatorsData, minVal, compLevel, optimalStakeMultiplier, rng, validatorPerformance)
	if len(scoreData.normalisedScores) == 0 {
		return scoreData.normalisedScores
	}
	validatorScoreEventSlice := make([]events.Event, 0, len(scoreData.normalisedScores))
	for _, k := range scoreData.nodeIDSlice {
		validatorScoreEventSlice = append(validatorScoreEventSlice, events.NewValidatorScore(ctx, k, epochSeq, scoreData.valScores[k], scoreData.normalisedScores[k],
			scoreData.rawValScores[k], scoreData.performanceScores[k]))
	}

	broker.SendBatch(validatorScoreEventSlice)
	return scoreData.normalisedScores
}

// calculate the validator score.
func calcValidatorScore(valStake, totalStake, minVal, compLevel, numVal, optimalStakeMultiplier num.Decimal) num.Decimal {
	a := num.MaxD(minVal, numVal.Div(compLevel))
	optStake := totalStake.Div(a)
	penaltyFlatAmt := num.MaxD(num.DecimalZero(), valStake.Sub(optStake))
	penaltyDownAmt := num.MaxD(num.DecimalZero(), valStake.Sub(optimalStakeMultiplier.Mul(optStake)))
	linearScore := valStake.Sub(penaltyFlatAmt).Sub(penaltyDownAmt).Div(totalStake) // totalStake guaranteed to be non zero at this point
	decimal1, _ := num.DecimalFromString("1.0")
	linearScore = num.MinD(decimal1, num.MaxD(num.DecimalZero(), linearScore))
	return linearScore
}

// calculate the total amount of tokens delegated to the validators including self and party delegation.
func calcTotalStake(validatorsData []*types.ValidatorData) *num.Uint {
	total := num.Zero()
	for _, d := range validatorsData {
		total.AddSum(d.StakeByDelegators, d.SelfStake)
	}
	return total
}
