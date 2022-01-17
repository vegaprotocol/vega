package rewards

import (
	"context"
	"math/rand"
	"sort"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

// Utilities for calculating delegation based rewards

var minThresholdDelegatorReward, _ = num.DecimalFromString("0.001")

// distribute rewards for a given asset account with the given settings of delegation and reward constraints.
func calculateRewards(epochSeq, asset, accountID string, rewardBalance *num.Uint, valScore map[string]num.Decimal, validatorDelegation []*types.ValidatorData, delegatorShare num.Decimal, maxPayout, minStakePerValidator *num.Uint, rng *rand.Rand, log *logging.Logger) *payout {
	minLeftOverForDistribution := num.Zero()
	if !maxPayout.IsZero() {
		minLeftOverForDistribution, _ = num.UintFromDecimal(minThresholdDelegatorReward.Mul(maxPayout.ToDecimal()))
	}

	// if there is no reward to give, return no payout
	rewards := map[string]*num.Uint{}
	totalRewardPayout := num.Zero()
	reward := rewardBalance.Clone()
	if reward.IsZero() {
		return &payout{
			partyToAmount: rewards,
			totalReward:   totalRewardPayout,
			asset:         asset,
			epochSeq:      epochSeq,
		}
	}

	for _, vd := range validatorDelegation {
		valScore := valScore[vd.NodeID] // normalised score
		if valScore.IsZero() {
			// if the validator isn't eligible for reward this round, nothing to do here
			continue
		}

		// how much reward is assigned to the validator and its delegators
		epochPayoutForValidatorAndDelegators := valScore.Mul(reward.ToDecimal())

		// calculate the fraction delegators to the validator get
		totalStakeForValidator := vd.StakeByDelegators.ToDecimal().Add(vd.SelfStake.ToDecimal())
		delegatorFraction := delegatorShare.Mul(vd.StakeByDelegators.ToDecimal()).Div(totalStakeForValidator) // totalStakeForValidator must be non zero as valScore is non zero
		validatorFraction := num.DecimalFromInt64(1).Sub(delegatorFraction)

		// if minStake is non zero and the validator has less total stake than required they don't get anything but their delegators still do
		if !minStakePerValidator.IsZero() && vd.SelfStake.LT(minStakePerValidator) {
			validatorFraction = num.DecimalZero()
		}

		// how much delegators take
		amountToGiveToDelegators, _ := num.UintFromDecimal(delegatorFraction.Mul(epochPayoutForValidatorAndDelegators))
		// calculate the potential reward for delegators and the validator
		amountToKeepByValidator, _ := num.UintFromDecimal(validatorFraction.Mul(epochPayoutForValidatorAndDelegators))

		log.Info("Rewards: reward calculation for validator",
			logging.String("epochSeq", epochSeq),
			logging.String("epochPayoutForValidatorAndDelegators", epochPayoutForValidatorAndDelegators.String()),
			logging.String("totalStakeForValidator", totalStakeForValidator.String()),
			logging.String("delegatorFraction", delegatorFraction.String()),
			logging.String("validatorFraction", validatorFraction.String()),
			logging.String("amountToGiveToDelegators", amountToGiveToDelegators.String()),
			logging.String("amountToKeepByValidator", amountToKeepByValidator.String()))

		// check how much reward the validator can accept with the cap per participant
		rewardForNode, ok := rewards[vd.NodeID]
		if !ok {
			rewardForNode = num.Zero()
		}
		// if there is no cap just add the total payout for the validator
		if maxPayout.IsZero() {
			rewards[vd.PubKey] = num.Zero().Add(rewardForNode, amountToKeepByValidator)
			totalRewardPayout.AddSum(amountToKeepByValidator)
		} else {
			balanceWithPayout := num.Zero().Add(rewardForNode, amountToKeepByValidator)
			if balanceWithPayout.LTE(maxPayout) {
				rewards[vd.PubKey] = balanceWithPayout
				totalRewardPayout.AddSum(amountToKeepByValidator)
			} else {
				rewards[vd.PubKey] = maxPayout
				totalRewardPayout.AddSum(num.Zero().Sub(maxPayout, rewardForNode))
			}
		}

		log.Info("Rewards: reward kept by validator for epoch (post max payout cap)", logging.String("epoch", epochSeq), logging.String("validator", vd.NodeID), logging.String("amountToKeepByValidator", rewards[vd.PubKey].String()))

		remainingRewardForDelegators := amountToGiveToDelegators

		// if there's nothing to give to delegators move on
		if remainingRewardForDelegators.IsZero() {
			continue
		}

		// calculate the weight of each delegator out of the delegated stake to the validator
		delegatorWeights := make(map[string]num.Decimal, len(vd.Delegators))
		weightSums := num.DecimalZero()
		decimalOne := num.DecimalFromInt64(1)
		for party, delegatorAmt := range vd.Delegators {
			delegatorWeight := delegatorAmt.ToDecimal().Div(vd.StakeByDelegators.ToDecimal()) // this is not entered if there are no delegators
			weightSums = weightSums.Add(delegatorWeight)
			delegatorWeights[party] = delegatorWeight
		}

		sortedParties := make([]string, 0, len(vd.Delegators))
		for party := range vd.Delegators {
			sortedParties = append(sortedParties, party)
		}
		sort.Strings(sortedParties)

		// NB: due to rounding errors this sum can be greater than 1
		// to avoid overflow, we choose at random a party adjust it by the error
		if weightSums.GreaterThan(decimalOne) {
			precisionError := weightSums.Sub(decimalOne)
			unluckyParty := sortedParties[rng.Intn(len(delegatorWeights))]
			delegatorWeights[unluckyParty] = num.MaxD(num.DecimalZero(), delegatorWeights[unluckyParty].Sub(precisionError))
		}

		// calculate delegator amounts
		// this may take a few rounds due to the cap on the reward a party can get
		// if we still have parties that haven't maxed their reward, they are split the remaining balance
		roundsRemaining := 10
		for {
			log.Info("Reward remaining to distribute to delegators", logging.String("epoch", epochSeq), logging.String("remainingRewardForDelegators", remainingRewardForDelegators.String()))

			totalAwardedThisRound := num.Zero()
			for _, party := range sortedParties {
				// check if the party has already rewards from other validators or previous rounds (this epoch)
				rewardForParty, ok := rewards[party]
				if !ok {
					rewardForParty = num.Zero()
				}

				delegatorWeight := delegatorWeights[party]
				rewardAsUint, _ := num.UintFromDecimal(delegatorWeight.Mul(remainingRewardForDelegators.ToDecimal()))
				if maxPayout.IsZero() {
					totalAwardedThisRound.AddSum(rewardAsUint)
					totalRewardPayout.AddSum(rewardAsUint)
					rewards[party] = num.Zero().Add(rewardForParty, rewardAsUint)
				} else {
					balanceWithPayout := num.Zero().Add(rewardForParty, rewardAsUint)
					if balanceWithPayout.LTE(maxPayout) {
						rewards[party] = balanceWithPayout
						totalAwardedThisRound.AddSum(rewardAsUint)
						totalRewardPayout.AddSum(rewardAsUint)
					} else {
						rewards[party] = maxPayout
						totalAwardedThisRound.AddSum(num.Zero().Sub(maxPayout, rewardForParty))
						totalRewardPayout.AddSum(num.Zero().Sub(maxPayout, rewardForParty))
					}
				}
			}
			roundsRemaining--

			// if we finished a round without distributing anything, we should stop
			// if this is the final round, stop
			// if the left over is too small for retrying to distribute, stop
			remainingRewardForDelegators = num.Zero().Sub(remainingRewardForDelegators, totalAwardedThisRound)
			if roundsRemaining == 0 || remainingRewardForDelegators.LT(minLeftOverForDistribution) || totalAwardedThisRound.IsZero() {
				break
			}
		}
	}

	return &payout{
		fromAccount:   accountID,
		partyToAmount: rewards,
		totalReward:   totalRewardPayout,
		asset:         asset,
		epochSeq:      epochSeq,
	}
}

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
