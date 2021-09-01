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
	t.Run("Unsupported registration of reward scheme fails", testRegisterRewardSchemeErr)
	t.Run("Unsupported update of reward scheme fails", testUpdateRewardSchemeErr)
	t.Run("Hardcoded registration of reward scheme for staking and delegation succeeds", testRegisterStakingAndDelegationRewardScheme)
	t.Run("Update asset for staking and delegation reward succeeds", testUpdateAssetForStakingAndDelegationRewardScheme)
	t.Run("Update asset for staking and delegation reward after max payout already set up succeeds", testUpdateAssetForStakingAndDelegationRewardSchemeWithMaxPayoutSetup)
	t.Run("Update max payout per participant for staking and delegation reward scheme succeeds", testUpdateMaxPayoutPerParticipantForStakingRewardScheme)
	t.Run("Update payout fraction for staking and delegation reward succeeds", testUpdatePayoutFractionForStakingRewardScheme)
	t.Run("Update payout delay for staking and delegation reward succeeds", testUpdatePayoutDelayForStakingRewardScheme)
	t.Run("Update delegator share for staking and delegation reward succeeds", testUpdateDelegatorShareForStakingRewardScheme)
	t.Run("Calculation of reward payout succeeds", testCalculateRewards)
	t.Run("Payout distribution succeeds", testDistributePayout)
	t.Run("Process epoch end to calculate payout with payout delay - all balance left on reward account is paid out", testOnEpochEndFullPayoutWithPayoutDelay)
	t.Run("Process epoch end to calculate payout with no delay - rewards are distributed successfully", testOnEpochEndNoPayoutDelay)
	t.Run("Process pending payouts on time update - time for payout hasn't come yet so no payouts sent", testOnChainTimeUpdateNoPayoutsToSend)
}

//test that registering reward scheme is unsupported
func testRegisterRewardSchemeErr(t *testing.T) {
	testEngine := getEngine(t)
	require.Error(t, ErrUnsupported, testEngine.engine.RegisterRewardScheme(&types.RewardScheme{}))
}

//test that updating reward scheme is unsupported
func testUpdateRewardSchemeErr(t *testing.T) {
	testEngine := getEngine(t)
	require.Error(t, ErrUnsupported, testEngine.engine.RegisterRewardScheme(&types.RewardScheme{}))
}

//test registration of hardcoded staking and delegation reward scheme
func testRegisterStakingAndDelegationRewardScheme(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()

	rs, ok := engine.rewardSchemes[stakingAndDelegationSchemeID]
	require.True(t, ok)
	require.Equal(t, rs.SchemeID, stakingAndDelegationSchemeID)
	require.Equal(t, types.RewardSchemeStakingAndDelegation, rs.Type)
	require.Equal(t, types.RewardSchemeScopeNetwork, rs.ScopeType)
	require.Equal(t, "", rs.Scope)
	require.Equal(t, 0, len(rs.Parameters))
	require.Equal(t, types.PayoutFractional, rs.PayoutType)
	require.Nil(t, rs.EndTime)
	require.Equal(t, 0, len(rs.RewardPoolAccountIDs))
}

//test updating of asset for staking and delegation reward which triggers the creation or get of the reward account for the asset
func testUpdateAssetForStakingAndDelegationRewardScheme(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()

	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")
	rs, ok := engine.rewardSchemes[stakingAndDelegationSchemeID]
	require.True(t, ok)
	require.Equal(t, 1, len(rs.RewardPoolAccountIDs))
	require.Equal(t, "!*ETH<", rs.RewardPoolAccountIDs[0])
}

//test updating of asset for staking and delegation reward which happens after max payout for asset has been updated
func testUpdateAssetForStakingAndDelegationRewardSchemeWithMaxPayoutSetup(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]
	rs.MaxPayoutPerAssetPerParty[""] = num.NewUint(10000)

	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")
	require.Equal(t, 1, len(rs.RewardPoolAccountIDs))
	require.Equal(t, "!*ETH<", rs.RewardPoolAccountIDs[0])
	require.Equal(t, 1, len(rs.MaxPayoutPerAssetPerParty))
	require.Equal(t, num.NewUint(10000), rs.MaxPayoutPerAssetPerParty["ETH"])
}

//test updating of max payout per participant for staking and delegation reward scheme
func testUpdateMaxPayoutPerParticipantForStakingRewardScheme(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]
	require.Equal(t, 0, len(rs.MaxPayoutPerAssetPerParty))

	engine.UpdateMaxPayoutPerParticipantForStakingRewardScheme(context.Background(), 10000)
	require.Equal(t, 1, len(rs.MaxPayoutPerAssetPerParty))
	require.Equal(t, num.NewUint(10000), rs.MaxPayoutPerAssetPerParty[""])
}

//test updading of payout fraction for staking and delegation reward scheme
func testUpdatePayoutFractionForStakingRewardScheme(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]
	require.Equal(t, 0.0, rs.PayoutFraction)

	engine.UpdatePayoutFractionForStakingRewardScheme(context.Background(), 0.1)
	require.Equal(t, 0.1, rs.PayoutFraction)
}

// test updating of payout delay for staking and delegation reward scheme
func testUpdatePayoutDelayForStakingRewardScheme(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]
	require.Equal(t, time.Duration(0), rs.PayoutDelay)

	engine.UpdatePayoutDelayForStakingRewardScheme(context.Background(), 1234*time.Second)
	require.Equal(t, 1234*time.Second, rs.PayoutDelay)
}

// test updating of payout delay for staking and delegation reward scheme
func testUpdateDelegatorShareForStakingRewardScheme(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]
	require.Equal(t, 0, len(rs.Parameters))

	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), 0.123456)
	require.Equal(t, 1, len(rs.Parameters))
	require.Equal(t, "delegatorShare", rs.Parameters["delegatorShare"].Name)
	require.Equal(t, "float", rs.Parameters["delegatorShare"].Type)
	require.Equal(t, "0.123456", rs.Parameters["delegatorShare"].Value)
}

// test calculation of reward payout
func testCalculateRewards(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), 0.3)
	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), 0)
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), 1.1)
	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	epoch := types.Epoch{}

	testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(testEngine.validatorData)

	res := engine.calculateRewards(context.Background(), "ETH", rs.RewardPoolAccountIDs[0], rs, num.NewUint(1000000), epoch)
	// node1, node2, node3, party1, party2
	require.Equal(t, 5, len(res.partyToAmount))

	require.Equal(t, num.NewUint(104571), res.partyToAmount["party1"])
	require.Equal(t, num.NewUint(24000), res.partyToAmount["party2"])
	require.Equal(t, num.NewUint(140000), res.partyToAmount["node1"])
	require.Equal(t, num.NewUint(400000), res.partyToAmount["node2"])
	require.Equal(t, num.NewUint(331428), res.partyToAmount["node3"])

	require.Equal(t, num.NewUint(999999), res.totalReward)
}

// test payout distribution
func testDistributePayout(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()

	// setup reward account
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), 0)
	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")

	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	// setup balance of reward account
	err := testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], num.NewUint(1000000))
	require.Nil(t, err)

	// setup general account for the party
	partyAccountID, err := testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "party1", "ETH")
	require.Nil(t, err)

	partyToAmount := map[string]*num.Uint{}
	partyToAmount["party1"] = num.NewUint(5000)

	payout := &payout{
		fromAccount:   rs.RewardPoolAccountIDs[0],
		totalReward:   num.NewUint(5000),
		partyToAmount: partyToAmount,
		asset:         "ETH",
	}

	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	engine.distributePayout(context.Background(), payout)

	rewardAccount, _ := engine.collateral.GetAccountByID(rs.RewardPoolAccountIDs[0])
	partyAccount, _ := engine.collateral.GetAccountByID(partyAccountID)

	require.Equal(t, num.NewUint(5000), partyAccount.Balance)
	require.Equal(t, num.NewUint(995000), rewardAccount.Balance)
}

// test on epoch end such that the full reward account balance can be reward with delay
func testOnEpochEndFullPayoutWithPayoutDelay(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	engine.UpdatePayoutFractionForStakingRewardScheme(context.Background(), 1.0)
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), 0.3)
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), 0)
	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), 1.1)

	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	//setup delay
	rs.PayoutDelay = 120 * time.Second

	//setup reward account balance
	err := testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], num.NewUint(1000000))
	require.Nil(t, err)

	// there is remaining 1000000 to distribute as payout
	epoch := types.Epoch{StartTime: time.Now(), EndTime: time.Now(), Seq: 1}
	engine.OnEpochEnd(context.Background(), epoch)

	// no account balances should change, no payouts made, just a pending reward scheme waiting to be processed after the delay
	epochEndPlusDelay := epoch.EndTime.Add(time.Second * 120)
	require.Equal(t, 1, len(engine.pendingPayouts[epochEndPlusDelay][epoch]))
	require.Equal(t, rs.SchemeID, engine.pendingPayouts[epochEndPlusDelay][epoch][0])

	// setup party accounts
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "party1", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "party2", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node1", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node2", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node3", "ETH")

	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(testEngine.validatorData)

	// setup another pending reward at a later time to observe that it remains pending after the current payout is made
	epoch2 := types.Epoch{StartTime: time.Now().Add(60 * time.Second), EndTime: time.Now().Add(90 * time.Second), Seq: 2}
	engine.OnEpochEnd(context.Background(), epoch2)

	// let time advance by 2 minutes
	engine.onChainTimeUpdate(context.Background(), epochEndPlusDelay)

	// the second reward is pending
	require.Equal(t, 1, len(engine.pendingPayouts))

	// get party account balances
	party1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party1", "ETH")
	party2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party2", "ETH")
	node1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node1", "ETH")
	node2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node2", "ETH")
	node3Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node3", "ETH")

	require.Equal(t, num.NewUint(104571), party1Acc.Balance)
	require.Equal(t, num.NewUint(24000), party2Acc.Balance)
	require.Equal(t, num.NewUint(140000), node1Acc.Balance)
	require.Equal(t, num.NewUint(400000), node2Acc.Balance)
	require.Equal(t, num.NewUint(331428), node3Acc.Balance)

	// advance to the end of the delay for the second reward + topup the balance of the reward account to be 1M again
	err = testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], num.NewUint(999999))
	require.Nil(t, err)

	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(testEngine.validatorData)
	engine.onChainTimeUpdate(context.Background(), epoch2.EndTime.Add(rs.PayoutDelay))

	// nothing is left pending
	require.Equal(t, 0, len(engine.pendingPayouts))

	party1Acc, _ = testEngine.collateral.GetPartyGeneralAccount("party1", "ETH")
	party2Acc, _ = testEngine.collateral.GetPartyGeneralAccount("party2", "ETH")
	node1Acc, _ = testEngine.collateral.GetPartyGeneralAccount("node1", "ETH")
	node2Acc, _ = testEngine.collateral.GetPartyGeneralAccount("node2", "ETH")
	node3Acc, _ = testEngine.collateral.GetPartyGeneralAccount("node3", "ETH")

	// expect balances to have doubled
	require.Equal(t, num.NewUint(104571*2), party1Acc.Balance)
	require.Equal(t, num.NewUint(24000*2), party2Acc.Balance)
	require.Equal(t, num.NewUint(140000*2), node1Acc.Balance)
	require.Equal(t, num.NewUint(400000*2), node2Acc.Balance)
	require.Equal(t, num.NewUint(331428*2), node3Acc.Balance)

}

// test payout distribution on epoch end with no delay
func testOnEpochEndNoPayoutDelay(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	engine.UpdatePayoutFractionForStakingRewardScheme(context.Background(), 1.0)
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), 0.3)
	engine.UpdateMinimumValidatorStakeForStakingRewardScheme(context.Background(), 0)
	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")
	engine.UpdateCompetitionLevelForStakingRewardScheme(context.Background(), 1.1)

	// setup party accounts
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "party1", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "party2", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node1", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node2", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node3", "ETH")

	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	//setup delay
	rs.PayoutDelay = 0 * time.Second

	//setup reward account balance
	err := testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], num.NewUint(1000000))
	require.Nil(t, err)

	// there is remaining 1000000 to distribute as payout
	epoch := types.Epoch{StartTime: time.Now(), EndTime: time.Now()}

	testEngine.delegation.EXPECT().ProcessEpochDelegations(gomock.Any(), gomock.Any()).Return(testEngine.validatorData)
	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	engine.OnEpochEnd(context.Background(), epoch)
	// total distributed is 999999
	require.Equal(t, 0, len(engine.pendingPayouts))

	// get party account balances
	party1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party1", "ETH")
	party2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party2", "ETH")
	node1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node1", "ETH")
	node2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node2", "ETH")
	node3Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node3", "ETH")

	require.Equal(t, num.NewUint(104571), party1Acc.Balance)
	require.Equal(t, num.NewUint(24000), party2Acc.Balance)
	require.Equal(t, num.NewUint(140000), node1Acc.Balance)
	require.Equal(t, num.NewUint(400000), node2Acc.Balance)
	require.Equal(t, num.NewUint(331428), node3Acc.Balance)
}

// test on time update - there are pending payouts but they are not yet due so nothing is paid or changed
func testOnChainTimeUpdateNoPayoutsToSend(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine

	engine.registerStakingAndDelegationRewardScheme()
	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")

	now := time.Now()
	payoutTime1 := now.Add(10 * time.Second)
	payoutTime2 := now.Add(20 * time.Second)

	engine.pendingPayouts[payoutTime1] = map[types.Epoch][]string{types.Epoch{}: []string{""}}
	engine.pendingPayouts[payoutTime2] = map[types.Epoch][]string{types.Epoch{}: []string{""}}

	testEngine.engine.onChainTimeUpdate(context.Background(), now)

	// expect no change to pending payouts as now is before the payout times
	require.Equal(t, 2, len(engine.pendingPayouts))
	require.Equal(t, 1, len(engine.pendingPayouts[payoutTime1]))
	require.Equal(t, 1, len(engine.pendingPayouts[payoutTime2]))
}

type testEngine struct {
	engine        *Engine
	ctrl          *gomock.Controller
	broker        *bmock.MockBroker
	epochEngine   *TestEpochEngine
	delegation    *mocks.MockDelegationEngine
	collateral    *collateral.Engine
	validatorData []*types.ValidatorData
}

func getEngine(t *testing.T) *testEngine {
	conf := NewDefaultConfig()
	ctrl := gomock.NewController(t)
	broker := bmock.NewMockBroker(ctrl)
	logger := logging.NewTestLogger()
	delegation := mocks.NewMockDelegationEngine(ctrl)
	epochEngine := &TestEpochEngine{callbacks: []func(context.Context, types.Epoch){}}
	ts := mocks.NewMockTimeService(ctrl)

	ts.EXPECT().GetTimeNow().AnyTimes()
	ts.EXPECT().NotifyOnTick(gomock.Any()).Times(1)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	collateral := collateral.New(logger, collateral.NewDefaultConfig(), broker, ts.GetTimeNow())
	asset := types.Asset{
		ID: "ETH",
		Details: &types.AssetDetails{
			Symbol: "ETH",
		},
	}

	collateral.EnableAsset(context.Background(), asset)

	engine := New(logger, conf, broker, delegation, epochEngine, collateral, ts)

	broker.EXPECT().Send(gomock.Any()).AnyTimes()

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
