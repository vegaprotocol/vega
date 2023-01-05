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
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	vmock "code.vegaprotocol.io/vega/core/validators/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var rng *rand.Rand

func init() {
	rng = rand.New(rand.NewSource(time.Now().Unix()))
}

func TestStakingRewards(t *testing.T) {
	t.Run("Calculate the reward when the balance of the reward account is 0", testCalcRewardNoBalance)
	t.Run("Calculate the reward when the validator scores are 0", testCalcRewardsZeroScores)
	t.Run("Reward is calculated correctly when max reward per participant is zero (i.e. unrestricted)", testCalcRewardsNoMaxPayout)
	t.Run("Reward is calculated correctly when max reward per participant restricted but not breached", testCalcRewardsMaxPayoutNotBreached)
	t.Run("Reward is calculated correctly when max reward per participant restricted and breached - no participant can be topped up", testCalcRewardSmallMaxPayoutBreached)
	t.Run("Reward is calculated correctly when max reward per participant restricted and breached - participant can be topped up", testCalcRewardsMaxPayoutBreachedPartyCanTakeMore)
	t.Run("Stop distributing leftover to delegation when remaining is less than 0.1% of max per participant", testEarlyStopCalcRewardsMaxPayoutBreachedPartyCanTakeMore)
}

func testCalcRewardNoBalance(t *testing.T) {
	delegatorShare, _ := num.DecimalFromString("0.3")
	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.UintZero(), map[string]num.Decimal{}, []*types.ValidatorData{}, delegatorShare, num.UintZero(), logging.NewTestLogger())
	require.Equal(t, num.UintZero(), res.totalReward)
	require.Equal(t, 0, len(res.partyToAmount))
}

func testCalcRewardsZeroScores(t *testing.T) {
	delegatorShare, _ := num.DecimalFromString("0.3")
	scores := map[string]num.Decimal{}
	scores["node1"] = num.DecimalZero()
	scores["node2"] = num.DecimalZero()
	scores["node3"] = num.DecimalZero()
	scores["node4"] = num.DecimalZero()

	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.NewUint(100000), scores, []*types.ValidatorData{}, delegatorShare, num.UintZero(), logging.NewTestLogger())
	require.Equal(t, num.UintZero(), res.totalReward)
	require.Equal(t, 0, len(res.partyToAmount))
}

func TestFilterZeros(t *testing.T) {
	delegatorShare, _ := num.DecimalFromString("0.3")
	scores := map[string]num.Decimal{}
	scores["node1"] = num.NewDecimalFromFloat(1)
	scores["node2"] = num.DecimalZero()
	scores["node3"] = num.DecimalZero()
	scores["node4"] = num.DecimalZero()

	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.NewUint(100000), scores, []*types.ValidatorData{{NodeID: "node1", PubKey: "node1", StakeByDelegators: num.NewUint(500), SelfStake: num.NewUint(1000), Delegators: map[string]*num.Uint{"zohar": num.UintZero(), "jeremy": num.NewUint(500)}}}, delegatorShare, num.UintZero(), logging.NewTestLogger())
	require.Equal(t, num.NewUint(100000), res.totalReward)
	require.Equal(t, 2, len(res.partyToAmount))
	_, ok := res.partyToAmount["zohar"]
	require.False(t, ok)

	_, ok = res.partyToAmount["jeremy"]
	require.True(t, ok)

	_, ok = res.partyToAmount["node1"]
	require.True(t, ok)
}

// nolint
func testCalcRewardsMaxPayoutRepsected(t *testing.T, maxPayout *num.Uint) {
	delegatorShare, _ := num.DecimalFromString("0.3")
	ctrl := gomock.NewController(t)
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	valPerformance := vmock.NewMockValidatorPerformance(ctrl)
	valPerformance.EXPECT().ValidatorPerformanceScore(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(1)).AnyTimes()

	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		PubKey:            "node1",
		SelfStake:         num.UintZero(),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		PubKey:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.UintZero(),
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
		SelfStake:         num.UintZero(),
		StakeByDelegators: num.UintZero(),
		Delegators:        map[string]*num.Uint{},
	}

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	valScores := map[string]num.Decimal{"node1": num.DecimalFromFloat(0.25), "node2": num.DecimalFromFloat(0.5), "node3": num.DecimalFromFloat(0.25), "node4": num.DecimalZero()}
	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, delegatorShare, maxPayout, logging.NewTestLogger())

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
	testCalcRewardsMaxPayoutRepsected(t, num.UintZero())
}

func testCalcRewardsMaxPayoutNotBreached(t *testing.T) {
	testCalcRewardsMaxPayoutRepsected(t, num.NewUint(1000000))
}

func testCalcRewardSmallMaxPayoutBreached(t *testing.T) {
	delegatorShare, _ := num.DecimalFromString("0.3")
	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		PubKey:            "node1",
		SelfStake:         num.UintZero(),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		PubKey:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.UintZero(),
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
		SelfStake:         num.UintZero(),
		StakeByDelegators: num.UintZero(),
		Delegators:        map[string]*num.Uint{},
	}

	ctrl := gomock.NewController(t)
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	valPerformance := vmock.NewMockValidatorPerformance(ctrl)
	valPerformance.EXPECT().ValidatorPerformanceScore(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(1)).AnyTimes()

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	valScores := map[string]num.Decimal{"node1": num.DecimalFromFloat(0.2), "node2": num.DecimalFromFloat(0.4), "node3": num.DecimalFromFloat(0.4), "node4": num.DecimalZero()}
	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, delegatorShare, num.NewUint(20000), logging.NewTestLogger())

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
	delegatorShare, _ := num.DecimalFromString("0.3")
	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		PubKey:            "node1",
		SelfStake:         num.UintZero(),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		PubKey:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.UintZero(),
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
		SelfStake:         num.UintZero(),
		StakeByDelegators: num.UintZero(),
		Delegators:        map[string]*num.Uint{},
	}

	ctrl := gomock.NewController(t)
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	valPerformance := vmock.NewMockValidatorPerformance(ctrl)
	valPerformance.EXPECT().ValidatorPerformanceScore(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(1)).AnyTimes()

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}

	valScores := map[string]num.Decimal{"node1": num.DecimalFromFloat(0.25), "node2": num.DecimalFromFloat(0.5), "node3": num.DecimalFromFloat(0.25), "node4": num.DecimalZero()}
	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, delegatorShare, num.NewUint(40000), logging.NewTestLogger())

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
	delegatorShare, _ := num.DecimalFromString("0.3")
	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		PubKey:            "node1",
		SelfStake:         num.UintZero(),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		PubKey:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.UintZero(),
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
		SelfStake:         num.UintZero(),
		StakeByDelegators: num.UintZero(),
		Delegators:        map[string]*num.Uint{},
	}

	ctrl := gomock.NewController(t)
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	valPerformance := vmock.NewMockValidatorPerformance(ctrl)
	valPerformance.EXPECT().ValidatorPerformanceScore(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(1)).AnyTimes()

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	valScores := map[string]num.Decimal{"node1": num.DecimalFromFloat(0.25), "node2": num.DecimalFromFloat(0.5), "node3": num.DecimalFromFloat(0.25), "node4": num.DecimalZero()}
	res := calculateRewardsByStake("1", "asset", "rewardsAccountID", num.NewUint(1000000), valScores, validatorData, delegatorShare, num.NewUint(1000000000), logging.NewTestLogger())

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
