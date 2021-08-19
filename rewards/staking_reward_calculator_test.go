package rewards

import (
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/require"
)

func TestStakingRewards(t *testing.T) {
	t.Run("Square root with 4 decimal places using only integer operations succeeds", testFourSquare)
	t.Run("Calculate correctly the validator score", testValidatorScore)
	t.Run("Calculate correctly the total delegate acorss all validators", testTotalDelegated)
	t.Run("Calculate normalised validator score", testCalcValidatorsScore)
	t.Run("Calculate the reward when the balance of the reward account is 0", testCalcRewardNoBalance)
	t.Run("Calculate the reward when the validator scores are 0", testCalcRewardsZeroScores)
	t.Run("Reward is calculated correctly when max reward per participant is zero (i.e. unrestricted)", testCalcRewardsNoMaxPayout)
	t.Run("Reward is calculated correctly when max reward per participant restricted but not breached", testCalcRewardsMaxPayoutNotBreached)
	t.Run("Reward is calculated correctly when max reward per participant restricted and breached - no participant can be topped up", testCalcRewardSmallMaxPayoutBreached)
	t.Run("Reward is calculated correctly when max reward per participant restricted and breached - participant can be topped up", testCalcRewardsMaxPayoutBreachedPartyCanTakeMore)
}

func testFourSquare(t *testing.T) {
	require.Equal(t, 4.0, foursqrt(16))
	require.Equal(t, 3.8729, foursqrt(15))
}

func testValidatorScore(t *testing.T) {
	validatorStake := 10000.0
	totalStake := 50000.0
	minVal := 5.0
	compLevel := 1.1

	// minVal > numVal/compLevel
	// valScore = sqrt(5 * 0.2 / 3) - sqrt(5 * 0.2 / 3)^3 = 0.5773 - 0.5773^3
	require.Equal(t, 0.384900175083, calcValidatorScore(validatorStake/totalStake, minVal, compLevel, 5.0))

	// minVal < numVal/compLevel
	// valScore = sqrt(9.0909090909 * 0.2 / 3) - sqrt(9.0909090909 * 0.2 / 3)^3 = 0.7784 - 0.7784^3
	require.Equal(t, 0.306762333696, calcValidatorScore(validatorStake/totalStake, minVal, compLevel, 10.0))
}

func testTotalDelegated(t *testing.T) {
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(10000),
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.Zero(),
	}
	validator3 := &types.ValidatorData{
		NodeID:            "node3",
		SelfStake:         num.NewUint(30000),
		StakeByDelegators: num.NewUint(40000),
	}
	require.Equal(t, num.NewUint(100000), calcTotalDelegated([]*types.ValidatorData{validator1, validator2, validator3}))
}

func testCalcValidatorsScore(t *testing.T) {
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(10000),
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.Zero(),
	}
	validator3 := &types.ValidatorData{
		NodeID:            "node3",
		SelfStake:         num.NewUint(30000),
		StakeByDelegators: num.NewUint(40000),
	}

	validator4 := &types.ValidatorData{
		NodeID:            "node4",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.Zero(),
	}

	valScores := calcValidatorsNormalisedScore([]*types.ValidatorData{validator1, validator2, validator3, validator4}, 5.0, 1.1)
	require.Equal(t, 4, len(valScores))

	//a = 5
	//normalisedStake = 0.1
	//scoreVal = sqrt(5 * 0.1 / 3) - sqrt(5 * 0.1 / 3)^3
	require.Equal(t, 0.34018276063200004/0.725082935715, valScores["node1"])

	//a = 5
	//normalisedStake = 0.2
	//scoreVal = sqrt(5 * 0.2 / 3) - sqrt(5 * 0.2 / 3)^3
	require.Equal(t, 0.384900175083/0.725082935715, valScores["node2"])

	//a = 5
	//normalisedStake = 0.7
	//scoreVal = sqrt(5 * 0.7 / 3) - sqrt(5 * 0.7 / 3)^3 => note that the score is actually negative
	require.Equal(t, 0.0, valScores["node3"])

	//a = 5
	//normalisedStake = 0
	//scoreVal = sqrt(5 / 3) - sqrt(5 / 3)^3 => note that the score is actually negative
	require.Equal(t, 0.0, valScores["node4"])
}

func testCalcRewardNoBalance(t *testing.T) {
	res := calculateRewards("asset", "rewardsAccountID", num.Zero(), map[string]float64{}, []*types.ValidatorData{}, 0.3, nil)
	require.Equal(t, num.Zero(), res.totalReward)
	require.Equal(t, 0, len(res.partyToAmount))
}

func testCalcRewardsZeroScores(t *testing.T) {
	scores := map[string]float64{}
	scores["node1"] = 0.0
	scores["node2"] = 0.0
	scores["node3"] = 0.0
	scores["node4"] = 0.0

	res := calculateRewards("asset", "rewardsAccountID", num.NewUint(100000), scores, []*types.ValidatorData{}, 0.3, nil)
	require.Equal(t, num.Zero(), res.totalReward)
	require.Equal(t, 0, len(res.partyToAmount))
}

func testCalcRewardsMaxPayoutRepsected(t *testing.T, maxPayout *num.Uint) {
	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	delegatorForVal3 := map[string]*num.Uint{}
	delegatorForVal3["party1"] = num.NewUint(40000)
	validator3 := &types.ValidatorData{
		NodeID:            "node3",
		SelfStake:         num.NewUint(30000),
		StakeByDelegators: num.NewUint(40000),
		Delegators:        delegatorForVal3,
	}

	validator4 := &types.ValidatorData{
		NodeID:            "node4",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	valScores := calcValidatorsNormalisedScore(validatorData, 5.0, 1.1)
	res := calculateRewards("asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, 0.3, maxPayout)

	// the normalised scores are as follows (from the test above)
	// node1 - 0.4691639313
	// node2 - 0.5308360687
	// node3 - 0
	// node4 - 0
	// as node3 and node4 has 0 score they get nothing.
	// given a reward of 1000000,
	// node1 and its delegators get 469,163
	// node2 and its delegators get 530,836
	// with a delegator share of 0.3,
	// for node1 as there's no self stake they get the full 0.3 share
	// for node1 the validator gets 0.7 of the reward for the node
	// for node2 there are no delegators so they get none
	// for node2 the validator gets 100% of the reward

	// node1, node2, party1, party2
	require.Equal(t, 4, len(res.partyToAmount))

	// 0.3 * 0.6 * 469163 = 84,449.34 => 84449
	require.Equal(t, num.NewUint(84449), res.partyToAmount["party1"])

	// 0.3 * 0.4 * 469163 = 56,299.56 => 56299
	require.Equal(t, num.NewUint(56299), res.partyToAmount["party2"])

	// 0.7 * 469163 = 328,414.1 => 328414
	require.Equal(t, num.NewUint(328414), res.partyToAmount["node1"])

	// 0.7 * 530836 = 371585 => 371585
	require.Equal(t, num.NewUint(371585), res.partyToAmount["node2"])

	require.Equal(t, num.NewUint(840747), res.totalReward)
}

func testCalcRewardsNoMaxPayout(t *testing.T) {
	testCalcRewardsMaxPayoutRepsected(t, num.Zero())
}

func testCalcRewardsMaxPayoutNotBreached(t *testing.T) {
	testCalcRewardsMaxPayoutRepsected(t, num.NewUint(1000000))
}

func testCalcRewardSmallMaxPayoutBreached(t *testing.T) {
	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	delegatorForVal3 := map[string]*num.Uint{}
	delegatorForVal3["party1"] = num.NewUint(40000)
	validator3 := &types.ValidatorData{
		NodeID:            "node3",
		SelfStake:         num.NewUint(30000),
		StakeByDelegators: num.NewUint(40000),
		Delegators:        delegatorForVal3,
	}

	validator4 := &types.ValidatorData{
		NodeID:            "node4",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	valScores := calcValidatorsNormalisedScore(validatorData, 5.0, 1.1)
	res := calculateRewards("asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, 0.3, num.NewUint(50000))

	// the normalised scores are as follows (from the test above)
	// node1 - 0.4691639313
	// node2 - 0.5308360687
	// node3 - 0
	// node4 - 0
	// as node3 and node4 has 0 score they get nothing.
	// given a reward of 1000000,
	// node1 and its delegators get 469,163
	// node2 and its delegators get 530,836
	// with a delegator share of 0.3,
	// for node1 as there's no self stake they get the full 0.3 share
	// for node1 the validator gets 0.7 of the reward for the node
	// for node2 there are no delegators so they get none
	// for node2 the validator gets 100% of the reward

	// node1, node2, party1, party2
	require.Equal(t, 4, len(res.partyToAmount))

	// 0.3 * 0.6 * 469163 = 84,449.34 => 84449 => with cap it becomes 50000
	require.Equal(t, num.NewUint(50000), res.partyToAmount["party1"])

	// 0.3 * 0.4 * 469163 = 56,299.56 => 56299 => with cap it becomes 50000
	require.Equal(t, num.NewUint(50000), res.partyToAmount["party2"])

	// 0.7 * 469163 = 328,414.1 => 328414 => with cap it becomes 50000
	require.Equal(t, num.NewUint(50000), res.partyToAmount["node1"])

	// 1 * 530836 = 530,836 => 530836 => with cap it becomes 50000
	require.Equal(t, num.NewUint(50000), res.partyToAmount["node2"])

	require.Equal(t, num.NewUint(200000), res.totalReward)
}

func testCalcRewardsMaxPayoutBreachedPartyCanTakeMore(t *testing.T) {
	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	delegatorForVal3 := map[string]*num.Uint{}
	delegatorForVal3["party1"] = num.NewUint(40000)
	validator3 := &types.ValidatorData{
		NodeID:            "node3",
		SelfStake:         num.NewUint(30000),
		StakeByDelegators: num.NewUint(40000),
		Delegators:        delegatorForVal3,
	}

	validator4 := &types.ValidatorData{
		NodeID:            "node4",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	valScores := calcValidatorsNormalisedScore(validatorData, 5.0, 1.1)
	res := calculateRewards("asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, 0.3, num.NewUint(80000))

	// the normalised scores are as follows (from the test above)
	// node1 - 0.4691639313
	// node2 - 0.5308360687
	// node3 - 0
	// node4 - 0
	// as node3 and node4 has 0 score they get nothing.
	// given a reward of 1000000,
	// node1 and its delegators get 469,163
	// node2 and its delegators get 530,836
	// with a delegator share of 0.3,
	// for node1 as there's no self stake they get the full 0.3 share
	// for node1 the validator gets 0.7 of the reward for the node
	// for node2 there are no delegators so they get none
	// for node2 the validator gets 100% of the reward

	// node1, node2, party1, party2
	require.Equal(t, 4, len(res.partyToAmount))

	// 0.3 * 0.6 * 469163 = 84,449.34 => 84449 => with cap it becomes 80000
	require.Equal(t, num.NewUint(80000), res.partyToAmount["party1"])

	// 0.3 * 0.4 * 469163 = 56,299.56 => 56299 + the remaining balance that can go to party => 60747
	require.Equal(t, num.NewUint(60747), res.partyToAmount["party2"])

	// 0.7 * 469163 = 328,414.1 => 328414 => with cap it becomes 80000
	require.Equal(t, num.NewUint(80000), res.partyToAmount["node1"])

	// 1 * 530836 = 530,836 => 530836 => with cap it becomes 80000
	require.Equal(t, num.NewUint(80000), res.partyToAmount["node2"])

	require.Equal(t, num.NewUint(300747), res.totalReward)
}
