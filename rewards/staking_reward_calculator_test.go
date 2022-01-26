package rewards

import (
	"context"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/rewards/mocks"
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
	t.Run("all validator score penalised to 0", testZeroValidatorScoresAllZero)
	t.Run("Calculate normalised validator score", testCalcValidatorsScore)
	t.Run("Calculate the reward when the balance of the reward account is 0", testCalcRewardNoBalance)
	t.Run("Calculate the reward when the validator scores are 0", testCalcRewardsZeroScores)
	t.Run("Reward is calculated correctly when max reward per participant is zero (i.e. unrestricted)", testCalcRewardsNoMaxPayout)
	t.Run("Reward is calculated correctly when max reward per participant restricted but not breached", testCalcRewardsMaxPayoutNotBreached)
	t.Run("Reward is calculated correctly when max reward per participant restricted and breached - no participant can be topped up", testCalcRewardSmallMaxPayoutBreached)
	t.Run("Reward is calculated correctly when max reward per participant restricted and breached - participant can be topped up", testCalcRewardsMaxPayoutBreachedPartyCanTakeMore)
	t.Run("Stop distributing leftover to delegation when remaining is less than 0.1% of max per participant", testEarlyStopCalcRewardsMaxPayoutBreachedPartyCanTakeMore)
}

func testValidatorScore(t *testing.T) {
	validatorStake := num.DecimalFromInt64(10000)
	largeValidatorStake := num.DecimalFromInt64(40000)
	extraLargeValidatorStake := num.DecimalFromInt64(60000)
	extraExtraLargeValidatorStake := num.DecimalFromInt64(70000)
	totalStake := num.DecimalFromInt64(100000.0)
	minVal := num.DecimalFromInt64(5)
	compLevel, _ := num.DecimalFromString("1.1")
	optimalStakeMultiplier, _ := num.DecimalFromString("3.0")

	// minVal > numVal/compLevel => a = 5
	// valScore = 0.1
	require.Equal(t, "0.10", calcValidatorScore(validatorStake, totalStake, minVal, compLevel, num.DecimalFromInt64(5), optimalStakeMultiplier).StringFixed(2))

	// // minVal < numVal/compLevel => a = 20
	// require.Equal(t, "0.05", calcValidatorScore(validatorStake, totalStake, minVal, compLevel, num.DecimalFromInt64(22), optimalStakeMultiplier).StringFixed(2))

	// minVal > numVal/compLevel => a = 5
	// valScore = 0.1
	// no pentalty
	require.Equal(t, "0.20", calcValidatorScore(largeValidatorStake, totalStake, minVal, compLevel, num.DecimalFromInt64(5), optimalStakeMultiplier).StringFixed(2))

	// minVal > numVal/compLevel => a = 5
	// valScore = 0.1
	// with flat pentalty
	require.Equal(t, "0.20", calcValidatorScore(extraLargeValidatorStake, totalStake, minVal, compLevel, num.DecimalFromInt64(5), optimalStakeMultiplier).StringFixed(2))

	// minVal > numVal/compLevel => a = 5
	// valScore = 0.1
	// with flat and down pentalty
	require.Equal(t, "0.10", calcValidatorScore(extraExtraLargeValidatorStake, totalStake, minVal, compLevel, num.DecimalFromInt64(5), optimalStakeMultiplier).StringFixed(2))
}

func testZeroValidatorScoresAllZero(t *testing.T) {
	ctrl := gomock.NewController(t)
	broker := bmock.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	validators := []*types.ValidatorData{}
	valPerformance := mocks.NewMockValidatorPerformance(ctrl)
	valPerformance.EXPECT().ValidatorPerformanceScore(gomock.Any()).Return(num.DecimalFromFloat(1)).AnyTimes()

	for i := 0; i < 3; i++ {
		validators = append(validators, &types.ValidatorData{
			NodeID:            "node" + strconv.Itoa(i),
			PubKey:            "node" + strconv.Itoa(i),
			SelfStake:         num.Zero(),
			StakeByDelegators: num.NewUint(10000),
		})
	}
	// setting up that all 3 nodes get score of 0 because they are penalised for having too much stake given the expected stake per node = 30000/(3/0.1) = 1000 (they have 10k each)
	scores := calcValidatorsNormalisedScore(context.Background(), broker, "1", validators, num.DecimalFromFloat(5), num.DecimalFromFloat(0.1), num.DecimalFromFloat(1), nil, valPerformance)
	for _, v := range scores {
		require.True(t, v.IsZero())
	}
}

func testTotalDelegated(t *testing.T) {
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		PubKey:            "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(10000),
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		PubKey:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.Zero(),
	}
	validator3 := &types.ValidatorData{
		NodeID:            "node3",
		PubKey:            "node3",
		SelfStake:         num.NewUint(30000),
		StakeByDelegators: num.NewUint(40000),
	}
	require.Equal(t, num.NewUint(100000), calcTotalStake([]*types.ValidatorData{validator1, validator2, validator3}))
}

func testCalcValidatorsScore(t *testing.T) {
	ctrl := gomock.NewController(t)
	broker := bmock.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	valPerformance := mocks.NewMockValidatorPerformance(ctrl)
	valPerformance.EXPECT().ValidatorPerformanceScore(gomock.Any()).DoAndReturn(
		func(nodeID string) num.Decimal {
			// i.e. validators 0-9
			if len(nodeID) == 5 {
				i, _ := strconv.Atoi(string(nodeID[4]))
				return num.DecimalFromInt64(int64(i + 1)).Div(num.DecimalFromFloat(10))
			}
			// i.e. validators 10-12
			return num.DecimalFromFloat(1)
		}).AnyTimes()

	validators := []*types.ValidatorData{}

	for i := 0; i < 12; i++ {
		validators = append(validators, &types.ValidatorData{
			NodeID:            "node" + strconv.Itoa(i),
			PubKey:            "node" + strconv.Itoa(i),
			SelfStake:         num.Zero(),
			StakeByDelegators: num.NewUint(3700000),
		})
	}

	validators = append(validators, &types.ValidatorData{
		NodeID:            "node12",
		PubKey:            "node12",
		SelfStake:         num.NewUint(3000),
		StakeByDelegators: num.Zero(),
	})

	minVal := num.DecimalFromInt64(5)
	compLevel, _ := num.DecimalFromString("1.1")
	optimalStakeMultiplier, _ := num.DecimalFromString("3.0")
	valScores := calcValidatorsNormalisedScore(context.Background(), broker, "1", validators, minVal, compLevel, optimalStakeMultiplier, rng, valPerformance)
	require.Equal(t, 13, len(valScores))

	require.Equal(t, "0.013", valScores["node0"].StringFixed(3))  // rawValScore=0.083327703083125 performanceScores=0.1 valScore=0.0083327703083125 normalisedScore=0.0133318920477066
	require.Equal(t, "0.027", valScores["node1"].StringFixed(3))  // rawValScore=0.083327703083125 performanceScores=0.2 valScore=0.016665540616625 normalisedScore=0.0266637840954131
	require.Equal(t, "0.040", valScores["node2"].StringFixed(3))  // rawValScore=0.083327703083125 performanceScores=0.3 valScore=0.0249983109249375 normalisedScore=0.0399956761431197
	require.Equal(t, "0.053", valScores["node3"].StringFixed(3))  // rawValScore=0.083327703083125 performanceScores=0.4 valScore=0.03333108123325 normalisedScore=0.0533275681908262
	require.Equal(t, "0.067", valScores["node4"].StringFixed(3))  // rawValScore=0.083327703083125 performanceScores=0.5 valScore=0.0416638515415625 normalisedScore=0.0666594602385328
	require.Equal(t, "0.080", valScores["node5"].StringFixed(3))  // rawValScore=0.083327703083125 performanceScores=0.6 valScore=0.049996621849875 normalisedScore=0.0799913522862393
	require.Equal(t, "0.093", valScores["node6"].StringFixed(3))  // rawValScore=0.083327703083125 performanceScores=0.7 valScore=0.0583293921581875 normalisedScore=0.0933232443339459
	require.Equal(t, "0.107", valScores["node7"].StringFixed(3))  // rawValScore=0.083327703083125 performanceScores=0.8 valScore=0.0666621624665 normalisedScore=0.1066551363816524
	require.Equal(t, "0.120", valScores["node8"].StringFixed(3))  // rawValScore=0.083327703083125 performanceScores=0.9 valScore=0.0749949327748125 normalisedScore=0.119987028429359
	require.Equal(t, "0.133", valScores["node9"].StringFixed(3))  // rawValScore=0.083327703083125 performanceScores=1 valScore=0.083327703083125 normalisedScore=0.1333189204770655
	require.Equal(t, "0.133", valScores["node10"].StringFixed(3)) // rawValScore=0.083327703083125 performanceScores=1 valScore=0.083327703083125 normalisedScore=0.1333189204770655
	require.Equal(t, "0.133", valScores["node11"].StringFixed(3)) // rawValScore=0.083327703083125 performanceScores=1 valScore=0.083327703083125 normalisedScore=0.1333189204770655
	require.Equal(t, "0.000", valScores["node12"].StringFixed(3)) // rawValScore=0.0000675630024998 performanceScores=1 valScore=0.0000675630024998 normalisedScore=0.0001080964220084

	validators[12] = &types.ValidatorData{
		NodeID:            "node12",
		PubKey:            "node12",
		SelfStake:         num.NewUint(3000),
		StakeByDelegators: num.NewUint(19900),
	}
	valScores = calcValidatorsNormalisedScore(context.Background(), broker, "1", validators, minVal, compLevel, optimalStakeMultiplier, rng, valPerformance)
	require.Equal(t, "0.001", valScores["node12"].StringFixed(3))

	validators[12] = &types.ValidatorData{
		NodeID:            "node12",
		PubKey:            "node12",
		SelfStake:         num.NewUint(3000),
		StakeByDelegators: num.NewUint(919900),
	}
	valScores = calcValidatorsNormalisedScore(context.Background(), broker, "1", validators, minVal, compLevel, optimalStakeMultiplier, rng, valPerformance)
	require.Equal(t, "0.032", valScores["node12"].StringFixed(3))
}

func testCalcRewardNoBalance(t *testing.T) {
	delegatorShare, _ := num.DecimalFromString("0.3")
	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.Zero(), map[string]num.Decimal{}, []*types.ValidatorData{}, delegatorShare, num.Zero(), num.Zero(), rng, logging.NewTestLogger())
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

	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.NewUint(100000), scores, []*types.ValidatorData{}, delegatorShare, num.Zero(), num.Zero(), rng, logging.NewTestLogger())
	require.Equal(t, num.Zero(), res.totalReward)
	require.Equal(t, 0, len(res.partyToAmount))
}

//nolint
func testCalcRewardsMaxPayoutRepsected(t *testing.T, maxPayout *num.Uint) {
	minVal := num.DecimalFromInt64(5)
	compLevel, _ := num.DecimalFromString("1.1")
	optimalStakeMultiplier, _ := num.DecimalFromString("3.0")
	delegatorShare, _ := num.DecimalFromString("0.3")
	ctrl := gomock.NewController(t)
	broker := bmock.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	valPerformance := mocks.NewMockValidatorPerformance(ctrl)
	valPerformance.EXPECT().ValidatorPerformanceScore(gomock.Any()).Return(num.DecimalFromFloat(1)).AnyTimes()

	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		PubKey:            "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		PubKey:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	delegatorForVal3 := map[string]*num.Uint{}
	delegatorForVal3["party1"] = num.NewUint(40000)
	validator3 := &types.ValidatorData{
		NodeID:            "node3",
		PubKey:            "node3",
		SelfStake:         num.NewUint(30000),
		StakeByDelegators: num.NewUint(40000),
		Delegators:        delegatorForVal3,
	}

	validator4 := &types.ValidatorData{
		NodeID:            "node4",
		PubKey:            "node4",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	valScores := calcValidatorsNormalisedScore(context.Background(), broker, "1", validatorData, minVal, compLevel, optimalStakeMultiplier, rng, valPerformance)
	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, delegatorShare, maxPayout, num.Zero(), rng, logging.NewTestLogger())

	// the normalised scores are as follows (from the test above)
	// node1 - 0.25
	// node2 - 0.5
	// node3 - 0.25
	// node4 - 0
	// as node3 and node4 has 0 score they get nothing.
	// given a reward of 1000000,
	//
	// node1 and its delegators get 250,000
	// node2 and its delegators get 500,000
	// node3 and its delegators get 250,000
	// with a delegator share of 0.3,
	// delegators to node1 get 0.3 * 250000 = 75000
	// party1 gets 0.6 * 75000 = 45000
	// party2 gets 0.4 * 75000 = 30000
	// node1 gets 175000
	// node2 gets 1 * 500000 = 500000
	// delegators to node3 get 0.3 * 4/7 * 250000 = 68571
	// party1 gets 42857
	// node3 gets 1 - (0.3*4/7) = 207142

	// node1, node2, node3, party1, party2
	require.Equal(t, 5, len(res.partyToAmount))

	require.Equal(t, num.NewUint(87857), res.partyToAmount["party1"])
	require.Equal(t, num.NewUint(30000), res.partyToAmount["party2"])
	require.Equal(t, num.NewUint(175000), res.partyToAmount["node1"])
	require.Equal(t, num.NewUint(500000), res.partyToAmount["node2"])
	require.Equal(t, num.NewUint(207142), res.partyToAmount["node3"])

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
	optimalStakeMultiplier, _ := num.DecimalFromString("3.0")
	delegatorShare, _ := num.DecimalFromString("0.3")
	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		PubKey:            "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		PubKey:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	delegatorForVal3 := map[string]*num.Uint{}
	delegatorForVal3["party1"] = num.NewUint(40000)
	validator3 := &types.ValidatorData{
		NodeID:            "node3",
		PubKey:            "node3",
		SelfStake:         num.NewUint(30000),
		StakeByDelegators: num.NewUint(40000),
		Delegators:        delegatorForVal3,
	}

	validator4 := &types.ValidatorData{
		NodeID:            "node4",
		PubKey:            "node4",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	ctrl := gomock.NewController(t)
	broker := bmock.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	valPerformance := mocks.NewMockValidatorPerformance(ctrl)
	valPerformance.EXPECT().ValidatorPerformanceScore(gomock.Any()).Return(num.DecimalFromFloat(1)).AnyTimes()

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	valScores := calcValidatorsNormalisedScore(context.Background(), broker, "1", validatorData, minVal, compLevel, optimalStakeMultiplier, rng, valPerformance)
	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, delegatorShare, num.NewUint(20000), num.Zero(), rng, logging.NewTestLogger())

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
	optimalStakeMultiplier, _ := num.DecimalFromString("3.0")
	delegatorShare, _ := num.DecimalFromString("0.3")
	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		PubKey:            "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		PubKey:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	delegatorForVal3 := map[string]*num.Uint{}
	delegatorForVal3["party1"] = num.NewUint(40000)
	validator3 := &types.ValidatorData{
		NodeID:            "node3",
		PubKey:            "node3",
		SelfStake:         num.NewUint(30000),
		StakeByDelegators: num.NewUint(40000),
		Delegators:        delegatorForVal3,
	}

	validator4 := &types.ValidatorData{
		NodeID:            "node4",
		PubKey:            "node4",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	ctrl := gomock.NewController(t)
	broker := bmock.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	valPerformance := mocks.NewMockValidatorPerformance(ctrl)
	valPerformance.EXPECT().ValidatorPerformanceScore(gomock.Any()).Return(num.DecimalFromFloat(1)).AnyTimes()

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	valScores := calcValidatorsNormalisedScore(context.Background(), broker, "1", validatorData, minVal, compLevel, optimalStakeMultiplier, rng, valPerformance)
	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, delegatorShare, num.NewUint(40000), num.Zero(), rng, logging.NewTestLogger())

	// the normalised scores are as follows (from the test above)
	// node1 - 0.25
	// node2 - 0.5
	// node3 - 0.25
	// node4 - 0
	// as node3 and node4 has 0 score they get nothing.
	// given a reward of 1000000,
	//
	// node1 and its delegators get 250,000
	// node2 and its delegators get 500,000
	// node3 and its delegators get 250,000
	// with a delegator share of 0.3,
	// delegators to node1 get 0.3 * 250000 = 75000
	// party1 gets 0.6 * 75000 = 45000 -> 40000
	// party2 gets 0.4 * 75000 = 30000 -> party can take 5k more from what's left =>
	// when distributing the 5k leftover:
	// iteration 0: party2 gets 0.4 * 75000 = 30000 = 2000
	// iteration 1: party2 gets 0.4*5000 = 2000
	// iteration 2: party2 gets 0.4*3000 = 1200
	// iteration 3: party2 gets 0.4*1800 = 720
	// iteration 4: party2 gets 0.4*1080 = 432
	// iteration 5: party2 gets 0.4*648 = 259
	// iteration 6: party2 gets 0.4*388 = 155
	// iteration 7: party2 gets 0.4*233 = 93
	// iteration 8: party2 gets 0.4*140 = 56
	// iteration 9: party2 gets 0.4*84 = 34
	// this runs for 10 iteration and stops therefore:
	// and party 2 gets: 30000 + 2000 + 1200 + 720 +432 + 259 + 155 + 93 + 56 + 34 = 34949
	// node1 gets 175000 -> 40000
	// node2 gets 1 * 500000 = 500000 -> 40000
	// delegators to node3 get 0.3 * 4/7 * 250000 = 42857
	// party1 gets 42857 -> 40000
	// node3 gets 1 - (0.3*4/7) = 207142 -> 40000
	// node1, node2, party1, party2
	require.Equal(t, 5, len(res.partyToAmount))

	require.Equal(t, num.NewUint(40000), res.partyToAmount["party1"])
	require.Equal(t, num.NewUint(34949), res.partyToAmount["party2"])
	require.Equal(t, num.NewUint(40000), res.partyToAmount["node1"])
	require.Equal(t, num.NewUint(40000), res.partyToAmount["node2"])
	require.Equal(t, num.NewUint(40000), res.partyToAmount["node3"])
	require.Equal(t, num.NewUint(194949), res.totalReward)
}

func testEarlyStopCalcRewardsMaxPayoutBreachedPartyCanTakeMore(t *testing.T) {
	minVal := num.DecimalFromInt64(5)
	compLevel, _ := num.DecimalFromString("1.1")
	optimalStakeMultiplier, _ := num.DecimalFromString("3.0")
	delegatorShare, _ := num.DecimalFromString("0.3")
	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		PubKey:            "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		PubKey:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	delegatorForVal3 := map[string]*num.Uint{}
	delegatorForVal3["party1"] = num.NewUint(40000)
	validator3 := &types.ValidatorData{
		NodeID:            "node3",
		PubKey:            "node3",
		SelfStake:         num.NewUint(30000),
		StakeByDelegators: num.NewUint(40000),
		Delegators:        delegatorForVal3,
	}

	validator4 := &types.ValidatorData{
		NodeID:            "node4",
		PubKey:            "node4",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	ctrl := gomock.NewController(t)
	broker := bmock.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	valPerformance := mocks.NewMockValidatorPerformance(ctrl)
	valPerformance.EXPECT().ValidatorPerformanceScore(gomock.Any()).Return(num.DecimalFromFloat(1)).AnyTimes()

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	valScores := calcValidatorsNormalisedScore(context.Background(), broker, "1", validatorData, minVal, compLevel, optimalStakeMultiplier, rng, valPerformance)
	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, delegatorShare, num.NewUint(1000000000), num.Zero(), rng, logging.NewTestLogger())

	// 0.1% of 1000000000 = 1000000 - this test is demonstrating that regardless of the remaining balance to give to delegators is less than 0.1% of the max
	// payout per participant, we still run one round and then stop.

	// the normalised scores are as follows (from the test above)
	// node1 - 0.25
	// node2 - 0.5
	// node3 - 0.25
	// node4 - 0
	// as node3 and node4 has 0 score they get nothing.
	// given a reward of 1000000,
	//
	// node1 and its delegators get 250,000
	// node2 and its delegators get 500,000
	// node3 and its delegators get 250,000
	// with a delegator share of 0.3,
	// delegators to node1 get 0.3 * 250000 = 75000
	// party1 gets 0.6 * 75000 = 45000
	// party2 gets 0.4 * 75000 = 30000
	// node1 gets 175000
	// node2 gets 1 * 500000 = 500000
	// delegators to node3 get 0.3 * 4/7 * 250000 = 42857
	// party1 gets 42857
	// node3 gets 1 - (0.3*4/7) = 207142 -> 207142
	// node1, node2, party1, party2
	require.Equal(t, 5, len(res.partyToAmount))

	require.Equal(t, num.NewUint(87857), res.partyToAmount["party1"])
	require.Equal(t, num.NewUint(30000), res.partyToAmount["party2"])
	require.Equal(t, num.NewUint(175000), res.partyToAmount["node1"])
	require.Equal(t, num.NewUint(500000), res.partyToAmount["node2"])
	require.Equal(t, num.NewUint(207142), res.partyToAmount["node3"])
	require.Equal(t, num.NewUint(999999), res.totalReward)
}
