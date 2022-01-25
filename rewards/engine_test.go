package rewards

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/types/num"

	bmock "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/rewards/mocks"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	t.Run("Update max payout per participant for staking and delegation reward scheme succeeds", testUpdateMaxPayoutPerParticipantForStakingRewardScheme)
	t.Run("Calculation of reward payout succeeds", testCalculateRewards)
	t.Run("Calculation of reward payout succeeds, epoch reward amount is capped by the max", testCalculateRewardsCappedByMaxPerEpoch)
	t.Run("Payout distribution succeeds", testDistributePayout)
	t.Run("Process epoch end to calculate payout with no delay - rewards are distributed successfully", testOnEpochEventNoPayoutDelay)
	t.Run("test should update voting power", testShouldUpdateVotingPower)
	t.Run("test voting power calculation", testVotingPowerCalculation)
}

func testShouldUpdateVotingPower(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.UpdateAssetForStakingAndDelegation(context.Background(), "VEGA")
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), num.DecimalFromFloat(0.3))
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), num.DecimalFromFloat(1.1))
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(3))
	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.DecimalZero())

	// now try to for all i between 1 and 999 and expect to get nil as no update is required
	for i := 1; i < 1000; i++ {
		require.Nil(t, engine.EndOfBlock(int64(i)))
	}
	testEngine.delegation.EXPECT().GetValidatorData().Return(testEngine.validatorData)
	require.NotNil(t, engine.EndOfBlock(0))
	testEngine.delegation.EXPECT().GetValidatorData().Return(testEngine.validatorData)
	require.NotNil(t, engine.EndOfBlock(1000))
	testEngine.delegation.EXPECT().GetValidatorData().Return(testEngine.validatorData)
	engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 2})
	require.NotNil(t, engine.EndOfBlock(1001))
}

func testVotingPowerCalculation(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.UpdateAssetForStakingAndDelegation(context.Background(), "VEGA")
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), num.DecimalFromFloat(0.3))
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), num.DecimalFromFloat(1.1))
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(3))
	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.DecimalZero())

	engine.OnEpochEvent(context.Background(), types.Epoch{Seq: 1})
	delegatorForVal1 := map[string]*num.Uint{}
	delegatorForVal1["party1"] = num.NewUint(6000)
	delegatorForVal1["party2"] = num.NewUint(4000)
	validator1 := &types.ValidatorData{
		NodeID:            "node1",
		PubKey:            "node1",
		TmPubKey:          "node1",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.NewUint(10000),
		Delegators:        delegatorForVal1,
	}
	validator2 := &types.ValidatorData{
		NodeID:            "node2",
		PubKey:            "node2",
		TmPubKey:          "node2",
		SelfStake:         num.NewUint(20000),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	delegatorForVal3 := map[string]*num.Uint{}
	delegatorForVal3["party1"] = num.NewUint(40000)
	validator3 := &types.ValidatorData{
		NodeID:            "node3",
		PubKey:            "node3",
		TmPubKey:          "node3",
		SelfStake:         num.NewUint(30000),
		StakeByDelegators: num.NewUint(40000),
		Delegators:        delegatorForVal3,
	}

	validator4 := &types.ValidatorData{
		NodeID:            "node4",
		PubKey:            "node4",
		TmPubKey:          "node4",
		SelfStake:         num.Zero(),
		StakeByDelegators: num.Zero(),
		Delegators:        map[string]*num.Uint{},
	}

	validatorData := []*types.ValidatorData{validator1, validator2, validator3, validator4}
	testEngine.delegation.EXPECT().GetValidatorData().Return(validatorData)

	// the normalised scores are as follows (from the test above)
	// node1 - 0.25
	// node2 - 0.5
	// node3 - 0.25
	// node4 - 0 => 1
	res := engine.EndOfBlock(1)
	require.Equal(t, int64(2500), res[0].VotingPower)
	require.Equal(t, int64(5000), res[1].VotingPower)
	require.Equal(t, int64(2500), res[2].VotingPower)
	require.Equal(t, int64(1), res[3].VotingPower)
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
	engine := testEngine.engine
	engine.UpdateAssetForStakingAndDelegation(context.Background(), "VEGA")
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), num.DecimalFromFloat(0.3))
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), num.DecimalFromFloat(1.1))
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.DecimalFromFloat(5))
	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.DecimalZero())

	epoch := types.Epoch{EndTime: time.Now()}
	rewardAccount, err := testEngine.collateral.CreateOrGetAssetRewardPoolAccount(context.Background(), "VEGA")
	require.NoError(t, err)
	testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(testEngine.validatorData)
	err = testEngine.collateral.IncrementBalance(context.Background(), rewardAccount, num.NewUint(1000000))
	require.Nil(t, err)

	payouts := engine.calculateRewardPayouts(context.Background(), epoch)
	res := payouts[0]
	// node1, node2, node3, party1, party2
	require.Equal(t, 5, len(res.partyToAmount))

	require.Equal(t, num.NewUint(104571), res.partyToAmount["party1"])
	require.Equal(t, num.NewUint(24000), res.partyToAmount["party2"])
	require.Equal(t, num.NewUint(140000), res.partyToAmount["node1"])
	require.Equal(t, num.NewUint(400000), res.partyToAmount["node2"])
	require.Equal(t, num.NewUint(331428), res.partyToAmount["node3"])
	require.Equal(t, epoch.EndTime.UnixNano(), res.timestamp)
	require.Equal(t, num.NewUint(999999), res.totalReward)
}

// test calculation of reward payout where the amount for epoch is capped by the max net param.
func testCalculateRewardsCappedByMaxPerEpoch(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.UpdateAssetForStakingAndDelegation(context.Background(), "VEGA")
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), num.DecimalFromFloat(0.3))
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), num.DecimalFromFloat(1.1))
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.DecimalFromFloat(5))
	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.DecimalZero())

	rewardAccount, err := testEngine.collateral.CreateOrGetAssetRewardPoolAccount(context.Background(), "VEGA")
	require.Nil(t, err)
	err = testEngine.collateral.IncrementBalance(context.Background(), rewardAccount, num.NewUint(1000000))
	require.NoError(t, err)
	epoch := types.Epoch{}

	testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(testEngine.validatorData)
	payouts := engine.calculateRewardPayouts(context.Background(), epoch)
	res := payouts[0]

	// node1, node2, node3, party1, party2
	require.Equal(t, 5, len(res.partyToAmount))
	require.Equal(t, num.NewUint(104571), res.partyToAmount["party1"])
	require.Equal(t, num.NewUint(24000), res.partyToAmount["party2"])
	require.Equal(t, num.NewUint(140000), res.partyToAmount["node1"])
	require.Equal(t, num.NewUint(400000), res.partyToAmount["node2"])
	require.Equal(t, num.NewUint(331428), res.partyToAmount["node3"])

	require.Equal(t, num.NewUint(999999), res.totalReward)
}

// test payout distribution.
func testDistributePayout(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.UpdateAssetForStakingAndDelegation(context.Background(), "VEGA")
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.DecimalFromFloat(5))

	// setup balance of reward account
	rewardAccountID, err := testEngine.collateral.CreateOrGetAssetRewardPoolAccount(context.Background(), "VEGA")
	require.NoError(t, err)
	err = testEngine.collateral.IncrementBalance(context.Background(), rewardAccountID, num.NewUint(1000000))
	require.Nil(t, err)
	partyToAmount := map[string]*num.Uint{}
	partyToAmount["party1"] = num.NewUint(5000)

	payout := &payout{
		fromAccount:   rewardAccountID,
		totalReward:   num.NewUint(5000),
		partyToAmount: partyToAmount,
		asset:         "VEGA",
	}

	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	engine.distributePayout(context.Background(), payout)

	rewardAccount, _ := engine.collateral.GetAccountByID(rewardAccountID)
	partyAccount, err := testEngine.collateral.GetPartyGeneralAccount("party1", "VEGA")
	require.Nil(t, err)

	require.Equal(t, num.NewUint(5000), partyAccount.Balance)
	require.Equal(t, num.NewUint(995000), rewardAccount.Balance)
}

// test payout distribution on epoch end with no delay.
func testOnEpochEventNoPayoutDelay(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.UpdateAssetForStakingAndDelegation(context.Background(), "VEGA")
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), num.DecimalFromFloat(0.3))
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), num.NewDecimalFromFloat(0))
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), num.DecimalFromFloat(1.1))
	engine.UpdateMinValidatorsStakingRewardScheme(context.Background(), 5)
	engine.UpdateOptimalStakeMultiplierStakingRewardScheme(context.Background(), num.DecimalFromFloat(5))
	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), num.DecimalZero())

	// setup party accounts
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "party1", "VEGA")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "party2", "VEGA")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node1", "VEGA")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node2", "VEGA")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node3", "VEGA")

	// setup reward account balance
	rewardAccountID, err := testEngine.collateral.CreateOrGetAssetRewardPoolAccount(context.Background(), "VEGA")
	require.NoError(t, err)
	err = testEngine.collateral.IncrementBalance(context.Background(), rewardAccountID, num.NewUint(1000000))
	require.Nil(t, err)

	// there is remaining 1000000 to distribute as payout
	epoch := types.Epoch{StartTime: time.Now(), EndTime: time.Now()}

	testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(testEngine.validatorData)
	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	engine.OnEpochEvent(context.Background(), epoch)
	engine.onChainTimeUpdate(context.Background(), epoch.EndTime.Add(120*time.Second))

	// get party account balances
	party1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party1", "VEGA")
	party2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party2", "VEGA")
	node1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node1", "VEGA")
	node2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node2", "VEGA")
	node3Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node3", "VEGA")

	require.Equal(t, num.NewUint(104571), party1Acc.Balance)
	require.Equal(t, num.NewUint(24000), party2Acc.Balance)
	require.Equal(t, num.NewUint(140000), node1Acc.Balance)
	require.Equal(t, num.NewUint(400000), node2Acc.Balance)
	require.Equal(t, num.NewUint(331428), node3Acc.Balance)
}

type testEngine struct {
	engine        *Engine
	ctrl          *gomock.Controller
	broker        *bmock.MockBroker
	epochEngine   *TestEpochEngine
	delegation    *mocks.MockDelegation
	collateral    *collateral.Engine
	validatorData []*types.ValidatorData
}

func getEngine(t *testing.T) *testEngine {
	t.Helper()
	conf := NewDefaultConfig()
	ctrl := gomock.NewController(t)
	broker := bmock.NewMockBroker(ctrl)
	logger := logging.NewTestLogger()
	delegation := mocks.NewMockDelegation(ctrl)
	epochEngine := &TestEpochEngine{callbacks: []func(context.Context, types.Epoch){}}
	ts := mocks.NewMockTimeService(ctrl)

	ts.EXPECT().GetTimeNow().AnyTimes()
	ts.EXPECT().NotifyOnTick(gomock.Any()).Times(1)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	collateral := collateral.New(logger, collateral.NewDefaultConfig(), broker, ts.GetTimeNow())
	asset := types.Asset{
		ID: "VEGA",
		Details: &types.AssetDetails{
			Symbol: "VEGA",
		},
	}

	collateral.EnableAsset(context.Background(), asset)
	valPerformance := mocks.NewMockValidatorPerformance(ctrl)
	valPerformance.EXPECT().ValidatorPerformanceScore(gomock.Any()).Return(num.DecimalFromFloat(1)).AnyTimes()

	feesTracker := mocks.NewMockFeesTracker(ctrl)
	engine := New(logger, conf, broker, delegation, epochEngine, collateral, ts, valPerformance, feesTracker)

	broker.EXPECT().Send(gomock.Any()).AnyTimes()

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

	return &testEngine{
		engine:        engine,
		ctrl:          ctrl,
		broker:        broker,
		epochEngine:   epochEngine,
		delegation:    delegation,
		collateral:    collateral,
		validatorData: validatorData,
	}
}

type TestEpochEngine struct {
	callbacks []func(context.Context, types.Epoch)
}

func (e *TestEpochEngine) NotifyOnEpoch(f func(context.Context, types.Epoch)) {
	e.callbacks = append(e.callbacks, f)
}
