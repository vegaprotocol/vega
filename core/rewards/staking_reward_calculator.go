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

package rewards

import (
	"math/rand"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var minThresholdDelegatorReward, _ = num.DecimalFromString("0.001")

// distribute rewards for a given asset account with the given settings of delegation and reward constraints.
func calculateRewardsByStake(epochSeq, asset, accountID string, rewardBalance *num.Uint, valScore map[string]num.Decimal, validatorDelegation []*types.ValidatorData, delegatorShare num.Decimal, maxPayout *num.Uint, rng *rand.Rand, log *logging.Logger) *payout {
	minLeftOverForDistribution := num.UintZero()
	if !maxPayout.IsZero() {
		minLeftOverForDistribution, _ = num.UintFromDecimal(minThresholdDelegatorReward.Mul(maxPayout.ToDecimal()))
	}

	// if there is no reward to give, return no payout
	rewards := map[string]*num.Uint{}
	totalRewardPayout := num.UintZero()
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
			rewardForNode = num.UintZero()
		}
		// if there is no cap just add the total payout for the validator
		if maxPayout.IsZero() {
			rewards[vd.PubKey] = num.UintZero().Add(rewardForNode, amountToKeepByValidator)
			totalRewardPayout.AddSum(amountToKeepByValidator)
		} else {
			balanceWithPayout := num.UintZero().Add(rewardForNode, amountToKeepByValidator)
			if balanceWithPayout.LTE(maxPayout) {
				rewards[vd.PubKey] = balanceWithPayout
				totalRewardPayout.AddSum(amountToKeepByValidator)
			} else {
				rewards[vd.PubKey] = maxPayout
				totalRewardPayout.AddSum(num.UintZero().Sub(maxPayout, rewardForNode))
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
		sortedParties := make([]string, 0, len(vd.Delegators))

		for party, delegatorAmt := range vd.Delegators {
			delegatorWeight := delegatorAmt.ToDecimal().Div(vd.StakeByDelegators.ToDecimal()) // this is not entered if there are no delegators
			weightSums = weightSums.Add(delegatorWeight)
			// if the party has 0 delegation, ignore it
			if !delegatorAmt.IsZero() {
				delegatorWeights[party] = delegatorWeight
				sortedParties = append(sortedParties, party)
			}
		}
		sort.Strings(sortedParties)

		adjustWeights(delegatorWeights, weightSums, sortedParties, decimalOne, rng)

		// calculate delegator amounts
		// this may take a few rounds due to the cap on the reward a party can get
		// if we still have parties that haven't maxed their reward, they are split the remaining balance
		roundsRemaining := 10
		for {
			log.Info("Reward remaining to distribute to delegators", logging.String("epoch", epochSeq), logging.String("remainingRewardForDelegators", remainingRewardForDelegators.String()))

			totalAwardedThisRound := num.UintZero()
			for _, party := range sortedParties {
				// check if the party has already rewards from other validators or previous rounds (this epoch)
				rewardForParty, ok := rewards[party]
				if !ok {
					rewardForParty = num.UintZero()
				}

				delegatorWeight := delegatorWeights[party]
				rewardAsUint, _ := num.UintFromDecimal(delegatorWeight.Mul(remainingRewardForDelegators.ToDecimal()))
				if maxPayout.IsZero() {
					totalAwardedThisRound.AddSum(rewardAsUint)
					totalRewardPayout.AddSum(rewardAsUint)
					rewards[party] = num.UintZero().Add(rewardForParty, rewardAsUint)
				} else {
					balanceWithPayout := num.UintZero().Add(rewardForParty, rewardAsUint)
					if balanceWithPayout.LTE(maxPayout) {
						rewards[party] = balanceWithPayout
						totalAwardedThisRound.AddSum(rewardAsUint)
						totalRewardPayout.AddSum(rewardAsUint)
					} else {
						rewards[party] = maxPayout
						totalAwardedThisRound.AddSum(num.UintZero().Sub(maxPayout, rewardForParty))
						totalRewardPayout.AddSum(num.UintZero().Sub(maxPayout, rewardForParty))
					}
				}
			}
			roundsRemaining--

			// if we finished a round without distributing anything, we should stop
			// if this is the final round, stop
			// if the left over is too small for retrying to distribute, stop
			remainingRewardForDelegators = num.UintZero().Sub(remainingRewardForDelegators, totalAwardedThisRound)
			if roundsRemaining == 0 || remainingRewardForDelegators.LT(minLeftOverForDistribution) || totalAwardedThisRound.IsZero() {
				break
			}
		}
	}

	if totalRewardPayout.GT(rewardBalance) {
		log.Error("The reward payout is greater than the reward balance, this should never happen", logging.String("reward-payout", totalRewardPayout.String()), logging.String("reward-balance", rewardBalance.String()))
	}

	return &payout{
		fromAccount:   accountID,
		partyToAmount: rewards,
		totalReward:   totalRewardPayout,
		asset:         asset,
		epochSeq:      epochSeq,
	}
}

func adjustWeights(delegatorWeights map[string]num.Decimal, weightSums num.Decimal, sortedParties []string, decimalOne num.Decimal, rng *rand.Rand) {
	// NB: due to rounding errors this sum can be greater than 1
	// to avoid overflow, we choose at random a party adjust it by the error
	// we keep looking until we find a candidate with sufficient weight to adjust by the precision error
	for weightSums.GreaterThan(decimalOne) {
		precisionError := weightSums.Sub(decimalOne)
		for precisionError.GreaterThan(num.DecimalZero()) {
			unluckyParty := sortedParties[rng.Intn(len(delegatorWeights))]
			if delegatorWeights[unluckyParty].LessThan(precisionError) {
				continue
			}
			delegatorWeights[unluckyParty] = num.MaxD(num.DecimalZero(), delegatorWeights[unluckyParty].Sub(precisionError))
			break
		}
		weightSums = num.DecimalZero()
		for _, d := range delegatorWeights {
			weightSums = weightSums.Add((d))
		}
	}
}
