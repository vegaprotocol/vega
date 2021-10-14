package rewards

import (
	"context"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	bmock "code.vegaprotocol.io/vega/broker/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var rng *rand.Rand

func init() {
	rng = rand.New(rand.NewSource(time.Now().Unix()))
}

func TestStakingRewards(t *testing.T) {
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

func testValidatorScore(t *testing.T) {
	validatorStake := num.DecimalFromInt64(10000)
	totalStake := num.DecimalFromInt64(100000.0)
	minVal := num.DecimalFromInt64(5)
	compLevel, _ := num.DecimalFromString("1.1")

	ratio := validatorStake.Div(totalStake)

	// minVal > numVal/compLevel => a = 5
	// valScore = 0.1
	require.Equal(t, "0.10", calcValidatorScore(ratio, minVal, compLevel, num.DecimalFromInt64(5)).StringFixed(2))

	// minVal < numVal/compLevel => a = 20
	require.Equal(t, "0.05", calcValidatorScore(ratio, minVal, compLevel, num.DecimalFromInt64(22)).StringFixed(2))
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
	ctrl := gomock.NewController(t)
	broker := bmock.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	validators := []*types.ValidatorData{}

	for i := 0; i < 12; i++ {
		validators = append(validators, &types.ValidatorData{
			NodeID:            "node" + strconv.Itoa(i),
			SelfStake:         num.Zero(),
			StakeByDelegators: num.NewUint(3700000),
		})
	}

	validators = append(validators, &types.ValidatorData{
		NodeID:            "node13",
		SelfStake:         num.NewUint(3000),
		StakeByDelegators: num.Zero(),
	})

	minVal := num.DecimalFromInt64(5)
	compLevel, _ := num.DecimalFromString("1.1")
	valScores := calcValidatorsNormalisedScore(context.Background(), broker, "1", validators, minVal, compLevel, rng)
	require.Equal(t, 13, len(valScores))

	for i := 0; i < 12; i++ {
		require.Equal(t, "0.083", valScores["node"+strconv.Itoa(i)].StringFixed(3))
	}
	require.Equal(t, "0.000", valScores["node13"].StringFixed(3))

	validators[12] = &types.ValidatorData{
		NodeID:            "node13",
		SelfStake:         num.NewUint(3000),
		StakeByDelegators: num.NewUint(19900),
	}
	valScores = calcValidatorsNormalisedScore(context.Background(), broker, "1", validators, minVal, compLevel, rng)
	require.Equal(t, "0.001", valScores["node13"].StringFixed(3))

	validators[12] = &types.ValidatorData{
		NodeID:            "node13",
		SelfStake:         num.NewUint(3000),
		StakeByDelegators: num.NewUint(919900),
	}
	valScores = calcValidatorsNormalisedScore(context.Background(), broker, "1", validators, minVal, compLevel, rng)
	require.Equal(t, "0.020", valScores["node13"].StringFixed(3))
}

func testCalcRewardNoBalance(t *testing.T) {
	delegatorShare, _ := num.DecimalFromString("0.3")
	res := calculateRewards("1", "asset", "rewardsAccountID", num.Zero(), map[string]num.Decimal{}, []*types.ValidatorData{}, delegatorShare, nil, num.Zero(), rng, logging.NewTestLogger())
	require.Equal(t, num.Zero(), res.totalReward)
	require.Equal(t, 0, len(res.partyToAmount))
}

func testCalcRewardsZeroScores(t *testing.T) {
	delegatorShare, _ := num.DecimalFromString("0.3")
	scores := map[string]num.Decimal{}
	scores["node1"] = num.DecimalZero()
	scores["node2"] = num.DecimalZero()
	scores["node3"] = num.DecimalZero()
	scores["node4"] = num.DecimalZero()

	res := calculateRewards("1", "asset", "rewardsAccountID", num.NewUint(100000), scores, []*types.ValidatorData{}, delegatorShare, nil, num.Zero(), rng, logging.NewTestLogger())
	require.Equal(t, num.Zero(), res.totalReward)
	require.Equal(t, 0, len(res.partyToAmount))
}

//nolint
func testCalcRewardsMaxPayoutRepsected(t *testing.T, maxPayout *num.Uint) {
	minVal := num.DecimalFromInt64(5)
	compLevel, _ := num.DecimalFromString("1.1")
	delegatorShare, _ := num.DecimalFromString("0.3")
	ctrl := gomock.NewController(t)
	broker := bmock.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

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
	valScores := calcValidatorsNormalisedScore(context.Background(), broker, "1", validatorData, minVal, compLevel, rng)
	res := calculateRewards("1", "asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, delegatorShare, maxPayout, num.Zero(), rng, logging.NewTestLogger())

	// the normalised scores are as follows (from the test above)
	// node1 - 0.2
	// node2 - 0.4
	// node3 - 0.4
	// node4 - 0
	// as node3 and node4 has 0 score they get nothing.
	// given a reward of 1000000,
	//
	// node1 and its delegators get 200,000
	// node2 and its delegators get 400,000
	// node3 and its delegators get 400,000
	// with a delegator share of 0.3,
	// delegators to node1 get 0.3 * 200000 = 60000
	// party1 gets 0.6 * 60000 = 36000
	// party2 gets 0.4 * 60000 = 24000
	// node1 gets 140000
	// node2 gets 1 * 400000 = 400000
	// delegators to node3 get 0.3 * 4/7 * 400000 = 68571
	// party1 gets 68571
	// node3 gets 1 - (0.3*4/7) = 331428

	// node1, node2, node3, party1, party2
	require.Equal(t, 5, len(res.partyToAmount))

	require.Equal(t, num.NewUint(104571), res.partyToAmount["party1"])
	require.Equal(t, num.NewUint(24000), res.partyToAmount["party2"])
	require.Equal(t, num.NewUint(140000), res.partyToAmount["node1"])
	require.Equal(t, num.NewUint(400000), res.partyToAmount["node2"])
	require.Equal(t, num.NewUint(331428), res.partyToAmount["node3"])

	require.Equal(t, num.NewUint(999999), res.totalReward)
}

func testCalcRewardsNoMaxPayout(t *testing.T) {
	testCalcRewardsMaxPayoutRepsected(t, num.Zero())
}

func testCalcRewardsMaxPayoutNotBreached(t *testing.T) {
	testCalcRewardsMaxPayoutRepsected(t, num.NewUint(1000000))
}

func testCalcRewardSmallMaxPayoutBreached(t *testing.T) {
	minVal := num.DecimalFromInt64(5)
	compLevel, _ := num.DecimalFromString("1.1")
	delegatorShare, _ := num.DecimalFromString("0.3")
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

	ctrl := gomock.NewController(t)
	broker := bmock.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	valScores := calcValidatorsNormalisedScore(context.Background(), broker, "1", validatorData, minVal, compLevel, rng)
	res := calculateRewards("1", "asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, delegatorShare, num.NewUint(20000), num.Zero(), rng, logging.NewTestLogger())

	// the normalised scores are as follows (from the test above)
	// node1 - 0.2
	// node2 - 0.4
	// node3 - 0.4
	// node4 - 0
	// as node3 and node4 has 0 score they get nothing.
	// given a reward of 1000000,
	//
	// node1 and its delegators get 200,000
	// node2 and its delegators get 400,000
	// node3 and its delegators get 400,000
	// with a delegator share of 0.3,
	// delegators to node1 get 0.3 * 200000 = 60000
	// party1 gets 0.6 * 60000 = 36000 -> 20000
	// party2 gets 0.4 * 60000 = 24000 -> 20000
	// node1 gets 140000 -> -> 20000
	// node2 gets 1 * 400000 = 400000 -> -> 20000
	// delegators to node3 get 0.3 * 4/7 * 400000 = 68571
	// party1 gets 68571 -> -> 20000
	// node3 gets 1 - (0.3*4/7) = 331428 -> -> 20000

	// node1, node2, node3, party1, party2
	require.Equal(t, 5, len(res.partyToAmount))

	require.Equal(t, num.NewUint(20000), res.partyToAmount["party1"])
	require.Equal(t, num.NewUint(20000), res.partyToAmount["party2"])
	require.Equal(t, num.NewUint(20000), res.partyToAmount["node1"])
	require.Equal(t, num.NewUint(20000), res.partyToAmount["node2"])
	require.Equal(t, num.NewUint(20000), res.partyToAmount["node3"])
	require.Equal(t, num.NewUint(100000), res.totalReward)
}

func testCalcRewardsMaxPayoutBreachedPartyCanTakeMore(t *testing.T) {
	minVal := num.DecimalFromInt64(5)
	compLevel, _ := num.DecimalFromString("1.1")
	delegatorShare, _ := num.DecimalFromString("0.3")
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

	ctrl := gomock.NewController(t)
	broker := bmock.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	valScores := calcValidatorsNormalisedScore(context.Background(), broker, "1", validatorData, minVal, compLevel, rng)
	res := calculateRewards("1", "asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, delegatorShare, num.NewUint(40000), num.Zero(), rng, logging.NewTestLogger())

	// the normalised scores are as follows (from the test above)
	// node1 - 0.2
	// node2 - 0.4
	// node3 - 0.4
	// node4 - 0
	// as node3 and node4 has 0 score they get nothing.
	// given a reward of 1000000,
	//
	// node1 and its delegators get 200,000
	// node2 and its delegators get 400,000
	// node3 and its delegators get 400,000
	// with a delegator share of 0.3,
	// delegators to node1 get 0.3 * 200000 = 60000
	// party1 gets 0.6 * 60000 = 36000 -> 36000
	// party2 gets 0.4 * 60000 = 24000 -> 24000
	// node1 gets 140000 -> 40000
	// node2 gets 1 * 400000 = 400000 -> 40000
	// delegators to node3 get 0.3 * 4/7 * 400000 = 68571
	// party1 gets 68571 -> 40000
	// node3 gets 1 - (0.3*4/7) = 331428 -> 40000
	// node1, node2, party1, party2
	require.Equal(t, 5, len(res.partyToAmount))

	require.Equal(t, num.NewUint(40000), res.partyToAmount["party1"])
	require.Equal(t, num.NewUint(24000), res.partyToAmount["party2"])
	require.Equal(t, num.NewUint(40000), res.partyToAmount["node1"])
	require.Equal(t, num.NewUint(40000), res.partyToAmount["node2"])
	require.Equal(t, num.NewUint(40000), res.partyToAmount["node3"])
	require.Equal(t, num.NewUint(184000), res.totalReward)
}
