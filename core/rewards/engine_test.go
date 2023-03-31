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
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/num"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/rewards/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	t.Run("Update max payout per participant for staking and delegation reward scheme succeeds", testUpdateMaxPayoutPerParticipantForStakingRewardScheme)
	t.Run("Calculation of reward payout succeeds", testCalculateRewards)
	t.Run("Calculation of reward payout succeeds with map per participant", testCalculateRewardsWithMaxPerParticipant)
	t.Run("Payout distribution succeeds", testDistributePayout)
	t.Run("Process epoch end to calculate payout with no delay - rewards are distributed successfully", testOnEpochEventNoPayoutDelay)
}

func TestRewardFactors(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine

	p, e := engine.calculateRewardFactors(num.DecimalFromInt64(10), num.DecimalFromInt64(10))
	require.Equal(t, "0.5", p.String())
	require.Equal(t, "0.5", e.String())

	p, e = engine.calculateRewardFactors(num.DecimalFromInt64(100), num.DecimalFromInt64(0))
	require.Equal(t, "1", p.String())
	require.Equal(t, "0", e.String())

	p, e = engine.calculateRewardFactors(num.DecimalFromInt64(0), num.DecimalFromInt64(1))
	require.Equal(t, "0", p.String())
	require.Equal(t, "1", e.String())

	p, e = engine.calculateRewardFactors(num.DecimalFromInt64(99999999), num.DecimalFromInt64(1))
	require.Equal(t, "0.99999999", p.String())
	require.Equal(t, "0.00000001", e.String())

	p, e = engine.calculateRewardFactors(num.DecimalFromInt64(1), num.DecimalFromInt64(99999999))
	require.Equal(t, "0.00000001", p.String())
	require.Equal(t, "0.99999999", e.String())
}

// test updating of max payout per participant for staking and delegation reward scheme.
func testUpdateMaxPayoutPerParticipantForStakingRewardScheme(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(10000))
	require.Equal(t, num.NewUint(10000), engine.global.maxPayoutPerParticipant)
}

// test calculation of reward payout.
func testCalculateRewards(t *testing.T) {
	testEngine := getEngine(t)
	now := time.Now()
	testEngine.timeService.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return now
		}).AnyTimes()

	engine := testEngine.engine
	engine.UpdateAssetForStakingAndDelegation(context.Background(), "VEGA")
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), num.DecimalFromFloat(0.3))
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), num.DecimalFromFloat(1.1))
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.DecimalFromFloat(5))
	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.DecimalZero())
	engine.UpdateErsatzRewardFactor(context.Background(), num.DecimalFromFloat(0.5))

	epoch := types.Epoch{EndTime: now}
	rewardAccount, err := testEngine.collateral.GetGlobalRewardAccount("VEGA")
	require.NoError(t, err)

	testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(testEngine.validatorData)
	testEngine.delegation.EXPECT().GetValidatorData().AnyTimes()
	testEngine.topology.EXPECT().RecalcValidatorSet(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	testEngine.topology.EXPECT().GetRewardsScores(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) (*types.ScoreData, *types.ScoreData) {
		return &types.ScoreData{
				NodeIDSlice: []string{"node1", "node2"},
				NormalisedScores: map[string]num.Decimal{
					"node1": num.DecimalFromFloat(0.2),
					"node2": num.DecimalFromFloat(0.8),
				},
			}, &types.ScoreData{
				NodeIDSlice: []string{"node3", "node4"},
				NormalisedScores: map[string]num.Decimal{
					"node3": num.DecimalFromFloat(0.6),
					"node4": num.DecimalFromFloat(0.4),
				},
			}
	})

	err = testEngine.collateral.IncrementBalance(context.Background(), rewardAccount.ID, num.NewUint(1000000))
	require.Nil(t, err)

	payouts := engine.calculateRewardPayouts(context.Background(), epoch)
	primary := payouts[0]
	ersatz := payouts[1]

	// calculation
	// node1 has total delegation of 15000
	// node2 has total delegation of 60000
	// node3 has total delegation of 4000
	// node4 has total delegation of 6000
	// primary validators have stake of 75000
	// ersatz validators have a stake of 10000
	// therefore primary get 0.9375 of the reward, ersatz 0.0625 of the reward
	// primary validators
	// node1, node2
	// node1 has normalised score of 0.2 => node1 and its delegators get 0.2 * 0.9375 * 1e6 = 1875000
	// out of 187500, delegators get 0.3 (delegatorShare) * 2/3 (the ratio of delegation by delegator in node1)= 37500
	// that leaves 187500-37500 = 150000 to node1
	// out of the 37500 party1 gets 0.6x (22500) and party2 gets 0.4x (15000) given their ratio of delegation in the node
	// node2 has normalised score of 0.8 => node 2 and its delegators get 0.8 * 0.9375 *1e6 = 750000
	// out of the 750000, delegators get 0.3 (delegatorShare) * 2/3 (the ratio of delegation by delegator in node2)= 150000
	// the 150000 goes exclusively to party1 and added to the 22500 they get from node1 to a total of 172500
	// node2 gets the rest of the 750000 => 600000
	// ersatz validators
	// node3 has normalised score of 0.6 => 0.6 * 62500 = 37500
	// node4 has normalised score of 0.4 => 0.4 * 62500 = 25000

	require.Equal(t, 4, len(primary.partyToAmount))

	require.Equal(t, num.NewUint(172500), primary.partyToAmount["party1"])
	require.Equal(t, num.NewUint(15000), primary.partyToAmount["party2"])
	require.Equal(t, num.NewUint(150000), primary.partyToAmount["node1"])
	require.Equal(t, num.NewUint(600000), primary.partyToAmount["node2"])
	require.Equal(t, num.NewUint(37500), ersatz.partyToAmount["node3"])
	require.Equal(t, num.NewUint(25000), ersatz.partyToAmount["node4"])
	require.Equal(t, epoch.EndTime.UnixNano(), primary.timestamp)
	require.Equal(t, epoch.EndTime.UnixNano(), ersatz.timestamp)
	require.Equal(t, num.NewUint(937500), primary.totalReward)
	require.Equal(t, num.NewUint(62500), ersatz.totalReward)
}

func testCalculateRewardsWithMaxPerParticipant(t *testing.T) {
	testEngine := getEngine(t)
	now := time.Now()
	testEngine.timeService.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return now
		}).AnyTimes()

	engine := testEngine.engine
	engine.UpdateAssetForStakingAndDelegation(context.Background(), "VEGA")
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), num.DecimalFromFloat(0.3))
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), num.DecimalFromFloat(1.1))
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.DecimalFromFloat(5))
	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.DecimalFromFloat(100000))
	engine.UpdateErsatzRewardFactor(context.Background(), num.DecimalFromFloat(0.5))

	epoch := types.Epoch{EndTime: now}
	rewardAccount, err := testEngine.collateral.GetGlobalRewardAccount("VEGA")
	require.NoError(t, err)
	testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(testEngine.validatorData)
	testEngine.delegation.EXPECT().GetValidatorData().AnyTimes()
	testEngine.topology.EXPECT().RecalcValidatorSet(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	testEngine.topology.EXPECT().GetRewardsScores(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) (*types.ScoreData, *types.ScoreData) {
		return &types.ScoreData{
				NodeIDSlice: []string{"node1", "node2"},
				NormalisedScores: map[string]num.Decimal{
					"node1": num.DecimalFromFloat(0.2),
					"node2": num.DecimalFromFloat(0.8),
				},
			}, &types.ScoreData{
				NodeIDSlice: []string{"node3", "node4"},
				NormalisedScores: map[string]num.Decimal{
					"node3": num.DecimalFromFloat(0.6),
					"node4": num.DecimalFromFloat(0.4),
				},
			}
	})

	err = testEngine.collateral.IncrementBalance(context.Background(), rewardAccount.ID, num.NewUint(1000000))
	require.Nil(t, err)

	payouts := engine.calculateRewardPayouts(context.Background(), epoch)
	primary := payouts[0]
	ersatz := payouts[1]

	// calculation
	// party1 should get 172500 => 100000
	// party2 should get 15000 => 15000
	// node1 should get 150000 => 100000
	// node2 should get 600000 => 100000
	// node3 should get 37500
	// node4 should get 25000

	require.Equal(t, 4, len(primary.partyToAmount))

	require.Equal(t, num.NewUint(100000), primary.partyToAmount["party1"])
	require.Equal(t, num.NewUint(15000), primary.partyToAmount["party2"])
	require.Equal(t, num.NewUint(100000), primary.partyToAmount["node1"])
	require.Equal(t, num.NewUint(100000), primary.partyToAmount["node2"])
	require.Equal(t, num.NewUint(37500), ersatz.partyToAmount["node3"])
	require.Equal(t, num.NewUint(25000), ersatz.partyToAmount["node4"])
	require.Equal(t, epoch.EndTime.UnixNano(), primary.timestamp)
	require.Equal(t, epoch.EndTime.UnixNano(), ersatz.timestamp)
	require.Equal(t, num.NewUint(315000), primary.totalReward)
	require.Equal(t, num.NewUint(62500), ersatz.totalReward)
}

// test payout distribution.
func testDistributePayout(t *testing.T) {
	testEngine := getEngine(t)
	now := time.Now()
	testEngine.timeService.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return now
		}).AnyTimes()

	engine := testEngine.engine
	engine.UpdateAssetForStakingAndDelegation(context.Background(), "VEGA")
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.DecimalFromFloat(5))
	engine.UpdateErsatzRewardFactor(context.Background(), num.DecimalFromFloat(0.5))

	// setup balance of reward account
	rewardAccount, err := testEngine.collateral.GetGlobalRewardAccount("VEGA")
	require.NoError(t, err)
	err = testEngine.collateral.IncrementBalance(context.Background(), rewardAccount.ID, num.NewUint(1000000))
	require.Nil(t, err)
	partyToAmount := map[string]*num.Uint{}
	partyToAmount["party1"] = num.NewUint(5000)

	payout := &payout{
		fromAccount:   rewardAccount.ID,
		totalReward:   num.NewUint(5000),
		partyToAmount: partyToAmount,
		asset:         "VEGA",
	}

	// testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	engine.distributePayout(context.Background(), payout)

	rewardAccount, _ = engine.collateral.GetAccountByID(rewardAccount.ID)
	partyAccount, err := testEngine.collateral.GetPartyGeneralAccount("party1", "VEGA")
	require.Nil(t, err)

	require.Equal(t, num.NewUint(5000), partyAccount.Balance)
	require.Equal(t, num.NewUint(995000), rewardAccount.Balance)
}

// test payout distribution on epoch end with no delay.
func testOnEpochEventNoPayoutDelay(t *testing.T) {
	testEngine := getEngine(t)
	now := time.Now()
	testEngine.timeService.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return now
		}).AnyTimes()

	engine := testEngine.engine
	engine.UpdateAssetForStakingAndDelegation(context.Background(), "VEGA")
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), num.DecimalFromFloat(0.3))
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), num.DecimalFromFloat(1.1))
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.DecimalFromFloat(5))
	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.DecimalZero())
	engine.UpdateErsatzRewardFactor(context.Background(), num.DecimalFromFloat(0.5))

	testEngine.delegation.EXPECT().GetValidatorData().AnyTimes()
	testEngine.topology.EXPECT().RecalcValidatorSet(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	testEngine.topology.EXPECT().GetRewardsScores(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) (*types.ScoreData, *types.ScoreData) {
		return &types.ScoreData{
				NodeIDSlice: []string{"node1", "node2"},
				NormalisedScores: map[string]num.Decimal{
					"node1": num.DecimalFromFloat(0.2),
					"node2": num.DecimalFromFloat(0.8),
				},
			}, &types.ScoreData{
				NodeIDSlice: []string{"node3", "node4"},
				NormalisedScores: map[string]num.Decimal{
					"node3": num.DecimalFromFloat(0.6),
					"node4": num.DecimalFromFloat(0.4),
				},
			}
	}).AnyTimes()

	// setup reward account balance
	rewardAccount, err := testEngine.collateral.GetGlobalRewardAccount("VEGA")
	require.NoError(t, err)
	err = testEngine.collateral.IncrementBalance(context.Background(), rewardAccount.ID, num.NewUint(1000000))
	require.Nil(t, err)

	// there is remaining 1000000 to distribute as payout
	epoch := types.Epoch{StartTime: now, EndTime: now}

	testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(testEngine.validatorData)
	engine.OnEpochEvent(context.Background(), epoch)

	// get party account balances
	party1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party1", "VEGA")
	party2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party2", "VEGA")
	node1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node1", "VEGA")
	node2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node2", "VEGA")
	node3Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node3", "VEGA")
	node4Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node4", "VEGA")

	require.Equal(t, num.NewUint(172500), party1Acc.Balance)
	require.Equal(t, num.NewUint(15000), party2Acc.Balance)
	require.Equal(t, num.NewUint(150000), node1Acc.Balance)
	require.Equal(t, num.NewUint(600000), node2Acc.Balance)
	require.Equal(t, num.NewUint(37500), node3Acc.Balance)
	require.Equal(t, num.NewUint(25000), node4Acc.Balance)
}

func TestErsatzTendermintRewardSplit(t *testing.T) {
	testEngine := getEngine(t)
	now := time.Now()
	testEngine.timeService.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return now
		}).AnyTimes()

	engine := testEngine.engine
	engine.UpdateAssetForStakingAndDelegation(context.Background(), "VEGA")
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), num.DecimalFromFloat(0.3))
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), num.DecimalFromFloat(1.1))
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.DecimalFromFloat(5))
	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.DecimalZero())
	engine.UpdateErsatzRewardFactor(context.Background(), num.DecimalFromFloat(0.5))

	testEngine.delegation.EXPECT().GetValidatorData().AnyTimes()
	testEngine.topology.EXPECT().RecalcValidatorSet(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	testEngine.topology.EXPECT().GetRewardsScores(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) (*types.ScoreData, *types.ScoreData) {
		return &types.ScoreData{
				NodeIDSlice: []string{"node1", "node2"},
				NormalisedScores: map[string]num.Decimal{
					"node1": num.DecimalFromFloat(0.2),
					"node2": num.DecimalFromFloat(0.8),
				},
			}, &types.ScoreData{
				NodeIDSlice: []string{"node3", "node4"},
				NormalisedScores: map[string]num.Decimal{
					"node3": num.DecimalFromFloat(0.6),
					"node4": num.DecimalFromFloat(0.4),
				},
			}
	}).AnyTimes()

	// setup reward account balance
	rewardAccount := testEngine.collateral.GetInfraFeeAccountIDs()[0]
	err := testEngine.collateral.IncrementBalance(context.Background(), rewardAccount, num.NewUint(1000000))
	require.Nil(t, err)

	// there is remaining 1000000 to distribute as payout
	epoch := types.Epoch{StartTime: now, EndTime: now}

	testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(testEngine.validatorData)
	engine.OnEpochEvent(context.Background(), epoch)

	// given the delegation breakdown we expect
	// tendermint validators to get 0.9375 x 1000000 => 937500
	// ersatzh validators to get => 0.0625 x 1000000 => 62500
	// in the tendermint validators node1 gets 0.2 x 937500 => 187500 and node2 gets 0.8 x 937500 => 750000
	// in the tendermint validators node3 gets 0.6 x 62500 => 37500 and node4 gets 0.4 x 62500 => 25000
	// from tendermint validators reward balance:
	// party1 gets 172500
	// party2 gets 15000
	// node1 gets 150000
	// node2 gets 600000
	// from ersatz validators reward balance:
	// node3 gets 37500
	// node 4 gets 25000

	// get party account balances
	party1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party1", "VEGA")
	party2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party2", "VEGA")
	node1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node1", "VEGA")
	node2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node2", "VEGA")
	node3Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node3", "VEGA")
	node4Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node4", "VEGA")

	require.Equal(t, num.NewUint(172500), party1Acc.Balance)
	require.Equal(t, num.NewUint(15000), party2Acc.Balance)
	require.Equal(t, num.NewUint(150000), node1Acc.Balance)
	require.Equal(t, num.NewUint(600000), node2Acc.Balance)
	require.Equal(t, num.NewUint(37500), node3Acc.Balance)
	require.Equal(t, num.NewUint(25000), node4Acc.Balance)
}

type testEngine struct {
	engine        *Engine
	ctrl          *gomock.Controller
	timeService   *mocks.MockTimeService
	broker        *bmocks.MockBroker
	epochEngine   *TestEpochEngine
	delegation    *mocks.MockDelegation
	collateral    *collateral.Engine
	validatorData []*types.ValidatorData
	topology      *mocks.MockTopology
}

func getEngine(t *testing.T) *testEngine {
	t.Helper()
	conf := NewDefaultConfig()
	ctrl := gomock.NewController(t)
	broker := bmocks.NewMockBroker(ctrl)
	logger := logging.NewTestLogger()
	delegation := mocks.NewMockDelegation(ctrl)
	epochEngine := &TestEpochEngine{
		callbacks: []func(context.Context, types.Epoch){},
		restore:   []func(context.Context, types.Epoch){},
	}
	ts := mocks.NewMockTimeService(ctrl)

	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	collateral := collateral.New(logger, collateral.NewDefaultConfig(), ts, broker)
	asset := types.Asset{
		ID: "VEGA",
		Details: &types.AssetDetails{
			Symbol: "VEGA",
		},
	}

	collateral.EnableAsset(context.Background(), asset)
	topology := mocks.NewMockTopology(ctrl)
	marketActivityTracker := mocks.NewMockMarketActivityTracker(ctrl)
	engine := New(logger, conf, broker, delegation, epochEngine, collateral, ts, marketActivityTracker, topology)

	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		PubKey:            "node1",
		SelfStake:         num.NewUint(5000),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	delegatorForVal2 := map[string]*num.Uint{}
	delegatorForVal2["party1"] = num.NewUint(40000)
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		PubKey:            "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.NewUint(40000),
		Delegators:        delegatorForVal2,
	}

	validator3 := &types.ValidatorData{
		NodeID:            "node3",
		PubKey:            "node3",
		SelfStake:         num.NewUint(4000),
		StakeByDelegators: num.UintZero(),
		Delegators:        map[string]*num.Uint{},
	}

	validator4 := &types.ValidatorData{
		NodeID:            "node4",
		PubKey:            "node4",
		SelfStake:         num.NewUint(6000),
		StakeByDelegators: num.UintZero(),
		Delegators:        map[string]*num.Uint{},
	}

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}

	return &testEngine{
		engine:        engine,
		ctrl:          ctrl,
		timeService:   ts,
		broker:        broker,
		epochEngine:   epochEngine,
		delegation:    delegation,
		collateral:    collateral,
		validatorData: validatorData,
		topology:      topology,
	}
}

type TestEpochEngine struct {
	callbacks []func(context.Context, types.Epoch)
	restore   []func(context.Context, types.Epoch)
}

func (e *TestEpochEngine) NotifyOnEpoch(f func(context.Context, types.Epoch), r func(context.Context, types.Epoch)) {
	e.callbacks = append(e.callbacks, f)
	e.restore = append(e.callbacks, r)
}

func (e *TestEpochEngine) GetTimeNow() time.Time {
	return time.Now()
}
