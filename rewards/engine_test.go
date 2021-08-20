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
	t.Run("Process epoch end to calculate payout with payout delay - no balance left on reward account including pending payouts", testOnEpochEndPendingPayoutZerosRewardAccountBalance)
	t.Run("Process epoch end to calculate payout with payout delay - some balance left on reward account including pending payouts", testOnEpochEndPendingPayoutRemainingRewardAccountBalance)
	t.Run("Process epoch end to calculate payout with payout delay - all balance left on reward account is paid out", testOnEpochEndFullPayoutWithPayoutDelay)
	t.Run("Process epoch end to calculte payout with no delay - rewards are distributed successfully", testOnEpochEndNoPayoutDelay)
	t.Run("Process pending payouts on time update - time for payout hasn't come yet so no payouts sent", testOnChainTimeUpdateNoPayoutsToSend)
	t.Run("Process pending payouts on time update - time for some payout has come but not for all", testOnChainTimeUpdateSomePayoutsToSend)
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

	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	epoch := types.Epoch{}

	testEngine.delegation.EXPECT().OnEpochEnd(gomock.Any(), gomock.Any(), gomock.Any()).Return(testEngine.validatorData)

	res := engine.calculateRewards(context.Background(), "ETH", rs.RewardPoolAccountIDs[0], rs, num.NewUint(1000000), epoch)
	// node1, node2, party1, party2
	require.Equal(t, 4, len(res.partyToAmount))

	// 0.3 * 0.6 * 469163 = 84,449.34 => 84449
	require.Equal(t, num.NewUint(84449), res.partyToAmount["party1"])

	// 0.3 * 0.4 * 469163 = 56,299.56 => 56299
	require.Equal(t, num.NewUint(56299), res.partyToAmount["party2"])

	// 0.7 * 469163 = 328,414.1 => 328414
	require.Equal(t, num.NewUint(328414), res.partyToAmount["node1"])

	// 1 * 530836 = 530,836 => 530836
	require.Equal(t, num.NewUint(530836), res.partyToAmount["node2"])

	require.Equal(t, num.NewUint(999998), res.totalReward)
}

// test payout distribution
func testDistributePayout(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()

	// setup reward account
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

	payout := &pendingPayout{
		fromAccount:   rs.RewardPoolAccountIDs[0],
		totalReward:   num.NewUint(5000),
		partyToAmount: partyToAmount,
		asset:         &types.Asset{ID: "ETH"},
	}

	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	engine.distributePayout(context.Background(), payout)

	rewardAccount, _ := engine.collateral.GetAccountByID(rs.RewardPoolAccountIDs[0])
	partyAccount, _ := engine.collateral.GetAccountByID(partyAccountID)

	require.Equal(t, num.NewUint(5000), partyAccount.Balance)
	require.Equal(t, num.NewUint(995000), rewardAccount.Balance)
}

// test on eopch end calculating reward such that there is nothing in the reward account when considering what's in the pending for payout
func testOnEpochEndPendingPayoutZerosRewardAccountBalance(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	engine.UpdatePayoutFractionForStakingRewardScheme(context.Background(), 1.0)
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), 0.3)
	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")

	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	//setup delay
	rs.PayoutDelay = 120 * time.Second

	//setup reward account balance
	err := testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], num.NewUint(1000))
	require.Nil(t, err)

	preEpochEnd := time.Now().Add(-1000 * time.Second)
	partyToAmt := map[string]*num.Uint{}
	partyToAmt["party1"] = num.NewUint(100)
	partyToAmt["party2"] = num.NewUint(200)
	partyToAmt["node1"] = num.NewUint(400)
	partyToAmt["node2"] = num.NewUint(300)

	existingPendingPayout := &pendingPayout{
		asset:         &types.Asset{ID: "ETH"},
		fromAccount:   rs.RewardPoolAccountIDs[0],
		partyToAmount: partyToAmt,
		totalReward:   num.NewUint(1000),
	}
	engine.pendingPayouts[preEpochEnd] = []*pendingPayout{existingPendingPayout}
	engine.rewardPoolToPendingPayoutBalance[existingPendingPayout.fromAccount] = num.NewUint(1000)

	// as the balance in the reward account is 1000 and we're using all of it, but there's pending payout of a 1000 so we expect nothing
	// to be paid out (or added to pending)
	engine.OnEpochEnd(context.Background(), types.Epoch{StartTime: time.Now(), EndTime: time.Now()})
	require.Equal(t, 1, len(engine.pendingPayouts))
	require.Equal(t, existingPendingPayout, engine.pendingPayouts[preEpochEnd][0])
}

// test on epoch end calculating reward such that the reward balance - pending payout still leaves some reward to pay and with delay the payout is added to pending
// and pending total is updated
func testOnEpochEndPendingPayoutRemainingRewardAccountBalance(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	engine.UpdatePayoutFractionForStakingRewardScheme(context.Background(), 1.0)
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), 0.3)
	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")

	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	//setup delay
	rs.PayoutDelay = 120 * time.Second

	//setup reward account balance
	err := testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], num.NewUint(1500000))
	require.Nil(t, err)

	preEpochPayoutTime := time.Now().Add(-1000 * time.Second)
	partyToAmt := map[string]*num.Uint{}
	partyToAmt["party1"] = num.NewUint(100)
	partyToAmt["party2"] = num.NewUint(200)
	partyToAmt["node1"] = num.NewUint(400)
	partyToAmt["node2"] = num.NewUint(300)

	existingPendingPayout := &pendingPayout{
		asset:         &types.Asset{ID: "ETH"},
		fromAccount:   rs.RewardPoolAccountIDs[0],
		partyToAmount: partyToAmt,
		totalReward:   num.NewUint(500000),
	}
	engine.pendingPayouts[preEpochPayoutTime] = []*pendingPayout{existingPendingPayout}
	engine.rewardPoolToPendingPayoutBalance[existingPendingPayout.fromAccount] = num.NewUint(500000)

	// there is remaining 100000 to distribute as payout
	epoch := types.Epoch{StartTime: time.Now(), EndTime: time.Now()}

	testEngine.delegation.EXPECT().OnEpochEnd(gomock.Any(), gomock.Any(), gomock.Any()).Return(testEngine.validatorData)

	engine.OnEpochEnd(context.Background(), epoch)
	// total pending is 1499998
	require.Equal(t, num.NewUint(1499998), engine.rewardPoolToPendingPayoutBalance[existingPendingPayout.fromAccount])
	require.Equal(t, 2, len(engine.pendingPayouts))
	epochEndPlusDelay := epoch.EndTime.Add(time.Second * 120)
	require.Equal(t, existingPendingPayout, engine.pendingPayouts[preEpochPayoutTime][0])
	require.Equal(t, num.NewUint(999998), engine.pendingPayouts[epochEndPlusDelay][0].totalReward)
	require.Equal(t, num.NewUint(84449), engine.pendingPayouts[epochEndPlusDelay][0].partyToAmount["party1"])
	require.Equal(t, num.NewUint(56299), engine.pendingPayouts[epochEndPlusDelay][0].partyToAmount["party2"])
	require.Equal(t, num.NewUint(328414), engine.pendingPayouts[epochEndPlusDelay][0].partyToAmount["node1"])
	require.Equal(t, num.NewUint(530836), engine.pendingPayouts[epochEndPlusDelay][0].partyToAmount["node2"])
}

// test on epoch end such that the full reward account balance can be reward with delay
func testOnEpochEndFullPayoutWithPayoutDelay(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	engine.UpdatePayoutFractionForStakingRewardScheme(context.Background(), 1.0)
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), 0.3)
	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")

	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	//setup delay
	rs.PayoutDelay = 120 * time.Second

	//setup reward account balance
	err := testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], num.NewUint(1000000))
	require.Nil(t, err)

	// there is remaining 1000000 to distribute as payout
	epoch := types.Epoch{StartTime: time.Now(), EndTime: time.Now()}

	testEngine.delegation.EXPECT().OnEpochEnd(gomock.Any(), gomock.Any(), gomock.Any()).Return(testEngine.validatorData)

	engine.OnEpochEnd(context.Background(), epoch)
	// total pending is 999998
	require.Equal(t, 1, len(engine.pendingPayouts))
	epochEndPlusDelay := epoch.EndTime.Add(time.Second * 120)
	resultPayout := engine.pendingPayouts[epochEndPlusDelay][0]
	require.Equal(t, num.NewUint(999998), engine.rewardPoolToPendingPayoutBalance[rs.RewardPoolAccountIDs[0]])
	require.Equal(t, num.NewUint(999998), resultPayout.totalReward)
	require.Equal(t, "ETH", resultPayout.asset.ID)
	require.Equal(t, rs.RewardPoolAccountIDs[0], resultPayout.fromAccount)
	require.Equal(t, num.NewUint(84449), resultPayout.partyToAmount["party1"])
	require.Equal(t, num.NewUint(56299), resultPayout.partyToAmount["party2"])
	require.Equal(t, num.NewUint(328414), resultPayout.partyToAmount["node1"])
	require.Equal(t, num.NewUint(530836), resultPayout.partyToAmount["node2"])
}

// test payout distribution on epoch end with no delay
func testOnEpochEndNoPayoutDelay(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	engine.UpdatePayoutFractionForStakingRewardScheme(context.Background(), 1.0)
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), 0.3)
	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")

	// setup party accounts
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "party1", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "party2", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node1", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node2", "ETH")

	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	//setup delay
	rs.PayoutDelay = 0 * time.Second

	//setup reward account balance
	err := testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], num.NewUint(1000000))
	require.Nil(t, err)

	// there is remaining 1000000 to distribute as payout
	epoch := types.Epoch{StartTime: time.Now(), EndTime: time.Now()}

	testEngine.delegation.EXPECT().OnEpochEnd(gomock.Any(), gomock.Any(), gomock.Any()).Return(testEngine.validatorData)
	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	engine.OnEpochEnd(context.Background(), epoch)
	// total distributed is 999998
	require.Equal(t, 0, len(engine.pendingPayouts))
	require.Equal(t, 0, len(engine.rewardPoolToPendingPayoutBalance))

	// get party account balances
	party1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party1", "ETH")
	party2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party2", "ETH")
	node1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node1", "ETH")
	node2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node2", "ETH")

	require.Equal(t, num.NewUint(84449), party1Acc.Balance)
	require.Equal(t, num.NewUint(56299), party2Acc.Balance)
	require.Equal(t, num.NewUint(328414), node1Acc.Balance)
	require.Equal(t, num.NewUint(530836), node2Acc.Balance)
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

	engine.pendingPayouts[payoutTime1] = []*pendingPayout{&pendingPayout{}}
	engine.pendingPayouts[payoutTime2] = []*pendingPayout{&pendingPayout{}}
	engine.rewardPoolToPendingPayoutBalance["rewardAccount"] = num.NewUint(100000)

	testEngine.engine.onChainTimeUpdate(context.Background(), now)

	// expect no change to pending payouts as now is before the payout times
	require.Equal(t, []*pendingPayout{&pendingPayout{}}, engine.pendingPayouts[payoutTime1])
	require.Equal(t, []*pendingPayout{&pendingPayout{}}, engine.pendingPayouts[payoutTime2])
	require.Equal(t, num.NewUint(100000), engine.rewardPoolToPendingPayoutBalance["rewardAccount"])
}

// test on time update - there are pending payouts but they are not yet due so nothing is paid or changed
func testOnChainTimeUpdateSomePayoutsToSend(t *testing.T) {
	testEngine := getEngine(t)
	engine := testEngine.engine
	engine.registerStakingAndDelegationRewardScheme()
	engine.UpdatePayoutFractionForStakingRewardScheme(context.Background(), 1.0)
	engine.UpdateDelegatorShareForStakingRewardScheme(context.Background(), 0.3)
	engine.UpdateAssetForStakingAndDelegationRewardScheme(context.Background(), "ETH")

	rs := engine.rewardSchemes[stakingAndDelegationSchemeID]

	err := testEngine.collateral.IncrementBalance(context.Background(), rs.RewardPoolAccountIDs[0], num.NewUint(1000000))
	require.Nil(t, err)

	// setup party accounts
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "party1", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "party2", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node1", "ETH")
	testEngine.collateral.CreatePartyGeneralAccount(context.Background(), "node2", "ETH")

	now := time.Now()
	payTime := now.Add(15 * time.Second)
	payoutTime1 := now.Add(10 * time.Second)
	payoutTime2 := now.Add(20 * time.Second)

	partyToAmt := map[string]*num.Uint{}
	partyToAmt["party1"] = num.NewUint(100)
	partyToAmt["party2"] = num.NewUint(200)
	partyToAmt["node1"] = num.NewUint(400)
	partyToAmt["node2"] = num.NewUint(300)

	payout1 := &pendingPayout{
		asset:         &types.Asset{ID: "ETH"},
		fromAccount:   rs.RewardPoolAccountIDs[0],
		partyToAmount: partyToAmt,
		totalReward:   num.NewUint(1000),
	}

	engine.pendingPayouts[payoutTime1] = []*pendingPayout{payout1}
	engine.pendingPayouts[payoutTime2] = []*pendingPayout{&pendingPayout{}}
	engine.rewardPoolToPendingPayoutBalance[payout1.fromAccount] = num.NewUint(1500)

	testEngine.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	testEngine.engine.onChainTimeUpdate(context.Background(), payTime)

	require.Equal(t, 1, len(engine.pendingPayouts))
	require.Equal(t, []*pendingPayout{&pendingPayout{}}, engine.pendingPayouts[payoutTime2])
	require.Equal(t, num.NewUint(500), engine.rewardPoolToPendingPayoutBalance[payout1.fromAccount])

	// get party account balances
	party1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party1", "ETH")
	party2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("party2", "ETH")
	node1Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node1", "ETH")
	node2Acc, _ := testEngine.collateral.GetPartyGeneralAccount("node2", "ETH")

	require.Equal(t, num.NewUint(100), party1Acc.Balance)
	require.Equal(t, num.NewUint(200), party2Acc.Balance)
	require.Equal(t, num.NewUint(400), node1Acc.Balance)
	require.Equal(t, num.NewUint(300), node2Acc.Balance)
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
