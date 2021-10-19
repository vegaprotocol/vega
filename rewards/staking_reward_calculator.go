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

func (e *Engine) calculatStakingAndDelegationRewards(ctx context.Context, broker Broker, epochSeq, asset, accountID string, rewardScheme *types.RewardScheme, rewardBalance *num.Uint, validatorData []*types.ValidatorData) *payout {
	delegatorShareStr := rewardScheme.Parameters["delegatorShare"].GetString()
	delegatorShare, err := num.DecimalFromString(delegatorShareStr)
	if err != nil {
		e.log.Panic("failed to read reward scheme param", logging.String("delegatorShare", rewardScheme.Parameters["delegatorShare"].Value))
	}

	compLevelStr := rewardScheme.Parameters["compLevel"].GetString()
	compLevel, err := num.DecimalFromString(compLevelStr)
	if err != nil {
		e.log.Panic("failed to read reward scheme param", logging.String("compLevel", rewardScheme.Parameters["compLevel"].Value))
	}

	// max payout is not mandatory, if it's not defined, pass nil so that max payout is not enforced for the asset
	maxPayoutPerParticipant, ok := rewardScheme.MaxPayoutPerAssetPerParty[asset]
	if !ok {
		maxPayoutPerParticipant = num.Zero()
	}

	minValStr := rewardScheme.Parameters["minVal"].GetString()
	minVal, err := num.DecimalFromString(minValStr)
	if err != nil {
		e.log.Panic("failed to read reward scheme param", logging.String("minVal", rewardScheme.Parameters["minVal"].Value))
	}

	// calculate the validator score for each validator and the total score for all
	validatorNormalisedScores := calcValidatorsNormalisedScore(ctx, broker, epochSeq, validatorData, minVal, compLevel, e.rng)
	for node, score := range validatorNormalisedScores {
		e.log.Info("Rewards: calculated normalised score", logging.String("validator", node), logging.String("normalisedScore", score.String()))
	}

	minStakePerValidator, err := rewardScheme.Parameters["minValStake"].GetUint()
	if err != nil {
		e.log.Panic("failed to read reward scheme param", logging.String("minValStake", rewardScheme.Parameters["minValStake"].Value))
	}

	maxPayoutPerEpoch, err := rewardScheme.Parameters["maxPayoutPerEpoch"].GetUint()
	if err != nil {
		e.log.Panic("failed to read reward scheme param", logging.String("maxPayoutPerEpoch", rewardScheme.Parameters["maxPayoutPerEpoch"].Value))
	}

	rewardBalance = num.Min(maxPayoutPerEpoch, rewardBalance)
	e.log.Info("Rewards: reward pot for for epoch with max payout per epoch", logging.String("epoch", epochSeq), logging.String("rewardBalance", rewardBalance.String()))

	// no point in doing anything after this point if the reward balance is 0
	if rewardBalance.IsZero() {
		return nil
	}
	return calculateRewards(epochSeq, asset, accountID, rewardBalance, validatorNormalisedScores, validatorData, delegatorShare, maxPayoutPerParticipant, minStakePerValidator, e.rng, e.log)
}

// distribute rewards for a given asset account with the given settings of delegation and reward constraints.
func calculateRewards(epochSeq, asset, accountID string, rewardBalance *num.Uint, valScore map[string]num.Decimal, validatorDelegation []*types.ValidatorData, delegatorShare num.Decimal, maxPayout, minStakePerValidator *num.Uint, rng *rand.Rand, log *logging.Logger) *payout {
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
		delegatorFraction := delegatorShare.Mul(vd.StakeByDelegators.ToDecimal()).Div(totalStakeForValidator)
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
			rewards[vd.NodeID] = num.Zero().Add(rewardForNode, amountToKeepByValidator)
			totalRewardPayout.AddSum(amountToKeepByValidator)
		} else {
			balanceWithPayout := num.Zero().Add(rewardForNode, amountToKeepByValidator)
			if balanceWithPayout.LTE(maxPayout) {
				rewards[vd.NodeID] = balanceWithPayout
				totalRewardPayout.AddSum(amountToKeepByValidator)
			} else {
				rewards[vd.NodeID] = maxPayout
				totalRewardPayout.AddSum(num.Zero().Sub(maxPayout, rewardForNode))
			}
		}

		log.Info("Rewards: reward kept by validator for epoch (post max payout cap)", logging.String("epoch", epochSeq), logging.String("validator", vd.NodeID), logging.String("amountToKeepByValidator", rewards[vd.NodeID].String()))

		remainingRewardForDelegators := amountToGiveToDelegators

		// calculate the weight of each delegator out of the delegated stake to the validator
		delegatorWeights := make(map[string]num.Decimal, len(vd.Delegators))
		weightSums := num.DecimalZero()
		decimalOne := num.DecimalFromInt64(1)
		for party, delegatorAmt := range vd.Delegators {
			delegatorWeight := delegatorAmt.ToDecimal().Div(vd.StakeByDelegators.ToDecimal())
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
		for !remainingRewardForDelegators.IsZero() {
			log.Info("Reward remaining to disrtibute to delegators", logging.String("epoch", epochSeq), logging.String("remainingRewardForDelegators", remainingRewardForDelegators.String()))

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
			// if we finished a round without distributing anything, we should stop
			if totalAwardedThisRound.IsZero() {
				break
			} else {
				// otherwise, update the available balance and repeat
				remainingRewardForDelegators = num.Zero().Sub(remainingRewardForDelegators, totalAwardedThisRound)
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

// calculate the score for each validator and normalise by the total score.
func calcValidatorsNormalisedScore(ctx context.Context, broker Broker, epochSeq string, validatorsData []*types.ValidatorData, minVal, compLevel num.Decimal, rng *rand.Rand) map[string]num.Decimal {
	// calculate the total amount of tokens delegated across all validators
	totalDelegated := calcTotalDelegated(validatorsData)
	totalScore := num.DecimalZero()
	rawScores := make(map[string]num.Decimal, len(validatorsData))
	valScores := make(map[string]num.Decimal, len(validatorsData))

	if totalDelegated.IsZero() {
		return valScores
	}

	// for each validator calculate the score
	nodeIDSlice := []string{}
	for _, vd := range validatorsData {
		totalValStake := num.Zero().Add(vd.StakeByDelegators, vd.SelfStake)
		normalisedValStake := totalValStake.ToDecimal().Div(totalDelegated.ToDecimal())
		valScore := calcValidatorScore(normalisedValStake, minVal, compLevel, num.DecimalFromInt64(int64(len(validatorsData))))
		rawScores[vd.NodeID] = valScore
		totalScore = totalScore.Add(valScore)
		nodeIDSlice = append(nodeIDSlice, vd.NodeID)
	}

	sort.Strings(nodeIDSlice)
	validatorScoreEventSlice := make([]events.Event, 0, len(valScores))

	scoreSum := num.DecimalZero()
	for _, k := range nodeIDSlice {
		score := rawScores[k]
		valScores[k] = score.Div(totalScore)
		scoreSum = scoreSum.Add(valScores[k])
	}

	// verify that the sum of scores is 1, if not adjust one score at random
	if scoreSum.GreaterThan(num.DecimalFromInt64(1)) {
		precisionError := scoreSum.Sub(num.DecimalFromInt64(1))
		unluckyValidator := rng.Intn(len(nodeIDSlice))
		valScores[nodeIDSlice[unluckyValidator]] = num.MaxD(valScores[nodeIDSlice[unluckyValidator]].Sub(precisionError), num.DecimalZero())
	}

	for _, k := range nodeIDSlice {
		validatorScoreEventSlice = append(validatorScoreEventSlice, events.NewValidatorScore(ctx, k, epochSeq, rawScores[k], valScores[k]))
	}

	broker.SendBatch(validatorScoreEventSlice)
	return valScores
}

// score_val(stake_val): min(1/a, validatorStake/totalStake).
func calcValidatorScore(normalisedValStake, minVal, compLevel, numVal num.Decimal) num.Decimal {
	a := num.MaxD(minVal, numVal.Div(compLevel))

	return num.MinD(normalisedValStake, num.DecimalFromInt64(1).Div(a))
}

// calculate the total amount of tokens delegated to the validators including self and party delegation.
func calcTotalDelegated(validatorsData []*types.ValidatorData) *num.Uint {
	total := num.Zero()
	for _, d := range validatorsData {
		total.AddSum(d.StakeByDelegators, d.SelfStake)
	}
	return total
}
