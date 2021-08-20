package rewards

import (
	"math"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

const (
	minVal    = 5.0
	compLevel = 1.1
)

func (e *Engine) calculatStakingAndDelegationRewards(asset string, accountID string, rewardScheme *types.RewardScheme, rewardBalance *num.Uint, validatorData []*types.ValidatorData) *pendingPayout {
	delegatorShare, err := rewardScheme.Parameters["delegatorShare"].GetFloat()
	if err != nil {
		e.log.Panic("failed to read reward scheme param", logging.String("delegatorShare", rewardScheme.Parameters["delegatorShare"].Value))
	}

	// max payout is not mandatory, if it's not defined, pass nil so that max payout is not enforced for the asset
	maxPayoutPerParticipant, ok := rewardScheme.MaxPayoutPerAssetPerParty[asset]
	if !ok {
		maxPayoutPerParticipant = num.Zero()
	}

	// calculate the validator score for each validator and the total score for all
	validatorNormalisedScores := calcValidatorsNormalisedScore(validatorData, minVal, compLevel)

	return calculateRewards(asset, accountID, rewardBalance, validatorNormalisedScores, validatorData, delegatorShare, maxPayoutPerParticipant)
}

// distribute rewards for a given asset account with the given settings of delegation and reward constraints
func calculateRewards(asset string, accountID string, rewardBalance *num.Uint, valScore map[string]float64, validatorDelegation []*types.ValidatorData, delegatorShare float64, maxPayout *num.Uint) *pendingPayout {
	// if there is no reward to give, return no payout
	rewards := map[string]*num.Uint{}
	totalRewardPayout := num.Zero()
	reward := rewardBalance.Clone()
	if reward.IsZero() {
		return &pendingPayout{
			partyToAmount: rewards,
			totalReward:   totalRewardPayout,
			asset:         asset,
		}
	}

	for _, vd := range validatorDelegation {
		valScore := valScore[vd.NodeID]
		if valScore == 0 {
			// if the validator isn't eligible for reward this round, nothing to do here
			continue
		}

		// how much reward is assigned to the validator and its delegators
		epochPayoutForValidatorAndDelegators := valScore * reward.Float64()

		// how much delegators take
		fractionDelegatorsGet := delegatorShare * vd.StakeByDelegators.Float64() / (vd.StakeByDelegators.Float64() + vd.SelfStake.Float64())

		// calculate the potential reward for delegators and the validator
		amountToKeepByValidator, _ := num.UintFromDecimal(num.NewDecimalFromFloat((1 - fractionDelegatorsGet) * epochPayoutForValidatorAndDelegators))

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

		amountToGiveToDelegators, _ := num.UintFromDecimal(num.NewDecimalFromFloat(fractionDelegatorsGet * epochPayoutForValidatorAndDelegators))
		remainingRewardForDelegators := amountToGiveToDelegators

		// calculate delegator amounts
		// this may take a few rounds due to the cap on the reward a party can get
		// if we still have parties that haven't maxed their reward, they are split the remaining balance
		for !remainingRewardForDelegators.IsZero() {
			totalAwardedThisRound := num.Zero()
			for party, delegatorAmt := range vd.Delegators {

				// check if the party has already rewards from other validators or previous rounds (this epoch)
				rewardForParty, ok := rewards[party]
				if !ok {
					rewardForParty = num.Zero()
				}

				rewardAsUint, _ := num.UintFromDecimal(num.NewDecimalFromFloat(delegatorAmt.Float64() * remainingRewardForDelegators.Float64() / vd.StakeByDelegators.Float64()))
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

	return &pendingPayout{
		fromAccount:   accountID,
		partyToAmount: rewards,
		totalReward:   totalRewardPayout,
		asset:         asset,
	}
}

// calculate the score for each validator and normalise by the total score
func calcValidatorsNormalisedScore(validatorsData []*types.ValidatorData, minVal, compLevel float64) map[string]float64 {
	// calculate the total amount of tokens delegated across all validators
	totalDelegated := calcTotalDelegated(validatorsData)
	totalScore := 0.0
	valScores := make(map[string]float64, len(validatorsData))
	// for each validator calculate the score
	for _, vd := range validatorsData {
		totalValStake := num.Zero().Add(vd.StakeByDelegators, vd.SelfStake)
		normalisedValStake := totalValStake.Float64() / totalDelegated.Float64()
		valScore := calcValidatorScore(normalisedValStake, minVal, compLevel, float64(len(validatorsData)))
		valScores[vd.NodeID] = valScore
		totalScore += valScore
	}

	for k, score := range valScores {
		valScores[k] = score / totalScore
	}

	return valScores
}

// score_val(stake_val): sqrt(a*stake_val/3)-(sqrt(a*stake_val/3)^3).
// To avoid issues with floating point computation, the sqrt function is computed to exactly four digits after the point.
// An example how this can be done using only integer calculations is in the example code.
/// Also, this function assumes that the stake is normalized, i.e., the sum of stake_val for all validators equals 1.
func calcValidatorScore(normalisedValStake, minVal, compLevel, numVal float64) float64 {
	a := math.Max(minVal, numVal/compLevel)
	x := foursqrt(a * normalisedValStake / 3.0)
	score := x - math.Pow(x, 3.0)
	if score < 0 {
		score = 0
	}
	return score
}

// Sqrt returns the square root of x.
// Based on code found in Hacker's Delight (Addison-Wesley, 2003):
// http://www.hackersdelight.org/
func iSqrt(x int) (r int) {
	if x < 0 {
		return -1
	}

	//Fast way to make p highest power of 4 <= x
	var n uint
	p := x
	if int64(p) >= 1<<32 {
		p >>= 32
		n = 32
	}
	if p >= 1<<16 {
		p >>= 16
		n += 16
	}
	if p >= 1<<8 {
		p >>= 8
		n += 8
	}
	if p >= 1<<4 {
		p >>= 4
		n += 4
	}
	if p >= 1<<2 {
		n += 2
	}
	p = 1 << n
	var b int
	for ; p != 0; p >>= 2 {
		b = r | p
		r >>= 1
		if x >= b {
			x -= b
			r |= p
		}
	}
	return
}

// Calculate the square root with 4 digits
// Use only integer manipualtions to do so.
func foursqrt(x float64) float64 {
	y := int(x * 10000 * 10000)
	s := iSqrt(y)
	return (float64(s) / 10000)
}

// calculate the total amount of tokens delegated to the validators including self and party delegation
func calcTotalDelegated(validatorsData []*types.ValidatorData) *num.Uint {
	total := num.Zero()
	for _, d := range validatorsData {
		total.AddSum(d.StakeByDelegators, d.SelfStake)
	}
	return total
}
