package collateral_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	ptypes "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testMarketID    = "7CPSHJB35AIQBTNMIE6NLFPZGHOYRQ3D"
	testMarketAsset = "BTC"
)

type testEngine struct {
	*collateral.Engine
	ctrl               *gomock.Controller
	broker             *mocks.MockBroker
	systemAccs         []*types.Account
	marketInsuranceID  string
	marketSettlementID string
}

type accEvt interface {
	events.Event
	Account() ptypes.Account
}

func TestCollateralTransfer(t *testing.T) {
	t.Run("test creating new - should create market accounts", testNew)
	t.Run("test collecting buys - both insurance and sufficient in trader accounts", testTransferLoss)
	t.Run("test collecting buys - trader account not empty, but insufficient", testTransferComplexLoss)
	t.Run("test collecting buys - trader missing some accounts", testTransferLossMissingTraderAccounts)
	t.Run("test collecting sells - cases where settle account is full + where insurance pool is tapped", testDistributeWin)
	t.Run("test collecting both buys and sells - Successfully collect buy and sell in a single call", testProcessBoth)
	t.Run("test distribution insufficient funds - Transfer losses (partial), distribute wins pro-rate", testProcessBothProRated)
	t.Run("test releas party margin account", testReleasePartyMarginAccount)
}

func TestCollateralMarkToMarket(t *testing.T) {
	t.Run("Mark to Market distribution, insufficient funcs - complex scenario", testProcessBothProRatedMTM)
	t.Run("Mark to Market successful", testMTMSuccess)
	t.Run("Mark to Market wins and losses do not match up, settlement not drained", testSettleBalanceNotZero)
}

func TestAddTraderToMarket(t *testing.T) {
	t.Run("Successful calls adding new traders (one duplicate, one actual new)", testAddTrader)
	t.Run("Can add a trader margin account if general account for asset exists", testAddMarginAccount)
	t.Run("Fail add trader margin account if no general account for asset exisrts", testAddMarginAccountFail)
}

func TestRemoveDistressed(t *testing.T) {
	t.Run("Successfully remove distressed trader and transfer balance", testRemoveDistressedBalance)
	t.Run("Successfully remove distressed trader, no balance transfer", testRemoveDistressedNoBalance)
}

func TestMarginUpdateOnOrder(t *testing.T) {
	t.Run("Successfully update margin on new order if general account balance is OK", testMarginUpdateOnOrderOK)
	t.Run("Successfully update margin on new order if general account balance is OK no shortfall with bond accound", testMarginUpdateOnOrderOKNotShortFallWithBondAccount)
	t.Run("Successfully update margin on new order if general account balance is OK will use bond account if exists", testMarginUpdateOnOrderOKUseBondAccount)
	t.Run("Successfully update margin on new order if general account balance is OK will use bond&general accounts if exists", testMarginUpdateOnOrderOKUseBondAndGeneralAccounts)
	t.Run("Successfully update margin on new order then rollback", testMarginUpdateOnOrderOKThenRollback)
	t.Run("Failed to update margin on new order if general account balance is OK", testMarginUpdateOnOrderFail)
}

func TestEnableAssets(t *testing.T) {
	t.Run("enable new asset - success", testEnableAssetSuccess)
	t.Run("enable new asset - failure duplicate", testEnableAssetFailureDuplicate)
	t.Run("create new account for bad asset - failure", testCreateNewAccountForBadAsset)
}

func TestBalanceTracking(t *testing.T) {
	t.Run("test a party without any account has no balance", testPartyWithoutAccountHasNoBalance)
	t.Run("test a party with an account has a balance", testPartyWithAccountHasABalance)
	t.Run("test a party with accounts cleared out has no balance", testPartyWithAccountsClearedOutHasNoBalance)
}

func TestCollateralContinuousTradingFeeTransfer(t *testing.T) {
	t.Run("Fees transfer continuous - no transfer", testFeesTransferContinuousNoTransfer)
	t.Run("fees transfer continuous - not funds", testFeeTransferContinuousNoFunds)
	t.Run("fees transfer continuous - not enough funds", testFeeTransferContinuousNotEnoughFunds)
	t.Run("fees transfer continuous - OK with enough in margin", testFeeTransferContinuousOKWithEnoughInMargin)
	t.Run("fees transfer continuous - OK with enough in general", testFeeTransferContinuousOKWithEnoughInGenral)
	t.Run("fees transfer continuous - OK with enough in margin + general", testFeeTransferContinuousOKWithEnoughInGeneralAndMargin)
	t.Run("fees transfer continuous - transfer with 0 amount", testFeeTransferContinuousOKWith0Amount)
	t.Run("fees transfer check account events", testFeeTransferContinuousOKCheckAccountEvents)
}

func TestCreateBondAccount(t *testing.T) {
	t.Run("create a bond account with success", testCreateBondAccountSuccess)
	t.Run("create a bond account with - failure no general account", testCreateBondAccountFailureNoGeneral)
}

func testPartyWithoutAccountHasNoBalance(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()

	party := "myparty"
	assert.False(t, eng.HasBalance(party))
}

func testPartyWithAccountHasABalance(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()

	party := "myparty"
	bal := num.NewUint(500)
	// create trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	acc, err := eng.Engine.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	assert.NoError(t, err)

	// then add some money
	err = eng.Engine.UpdateBalance(context.Background(), acc, bal)
	assert.Nil(t, err)

	evt := eng.broker.GetLastByTypeAndID(events.AccountEvent, acc)
	require.NotNil(t, evt)
	ae, ok := evt.(accEvt)
	require.True(t, ok)
	account := ae.Account()
	// balance of the account after all of the events should be 500
	require.Equal(t, bal.Uint64(), account.Balance)
	assert.True(t, eng.HasBalance(party))
}

func testPartyWithAccountsClearedOutHasNoBalance(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()

	party := "myparty"
	bal := num.NewUint(500)
	// create trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	acc, err := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	assert.NoError(t, err)

	// then add some money
	err = eng.UpdateBalance(context.Background(), acc, bal)
	assert.Nil(t, err)
	evt := eng.broker.GetLastByTypeAndID(events.AccountEvent, acc)
	require.NotNil(t, evt)
	ae, ok := evt.(accEvt)
	require.True(t, ok)
	account := ae.Account()
	// balance of the account after all of the events should be 500
	require.Equal(t, bal.Uint64(), account.Balance)

	// then add some money
	err = eng.DecrementBalance(context.Background(), acc, bal)
	assert.Nil(t, err)
	// get the last event after decrementing it
	// we don't have to perform a nil-check here because we know there is an event for this ID anyway
	evt = eng.broker.GetLastByTypeAndID(events.AccountEvent, acc)
	// this type assertion is guaranteed to work here anyway
	ae = evt.(accEvt)
	// balance should be zero
	require.Zero(t, ae.Account().Balance)

	// HasBalance returns true, seeing as zero is technically a balance...
	assert.True(t, eng.HasBalance(party))
}

func testCreateBondAccountFailureNoGeneral(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()

	trader := "mytrader"
	// create trader
	_, err := eng.CreatePartyBondAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.EqualError(t, err, "party general account missing when trying to create a bond account")
}

func testCreateBondAccountSuccess(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()

	trader := "mytrader"
	bal := num.NewUint(500)
	// create trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, err := eng.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	require.NoError(t, err)
	bnd, err := eng.CreatePartyBondAccount(context.Background(), trader, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.UpdateBalance(context.Background(), bnd, bal)
	assert.Nil(t, err)

	evt := eng.broker.GetLastByTypeAndID(events.AccountEvent, bnd)
	require.NotNil(t, evt)
	ae, ok := evt.(accEvt)
	require.True(t, ok)
	account := ae.Account()
	require.Equal(t, bal.Uint64(), account.Balance)
	// these two checks are a bit redundant at this point
	// but at least we're verifying that the GetAccountByID and the latest event return the same state
	bndacc, _ := eng.GetAccountByID(bnd)
	assert.Equal(t, account.Balance, bndacc.Balance.Uint64())
}

func testFeesTransferContinuousNoTransfer(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()

	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFees{})
	assert.Nil(t, transfers)
	assert.Nil(t, err)
}

func testReleasePartyMarginAccount(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()

	trader := "mytrader"
	// create trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	gen, err := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	require.NoError(t, err)

	mar, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(context.Background(), gen, num.NewUint(100))
	assert.Nil(t, err)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(context.Background(), mar, num.NewUint(500))
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	_, err = eng.ClearPartyMarginAccount(
		context.Background(), trader, testMarketID, testMarketAsset)
	assert.NoError(t, err)
	generalAcc, _ := eng.GetAccountByID(gen)
	assert.Equal(t, num.NewUint(600), generalAcc.Balance)
	marginAcc, _ := eng.GetAccountByID(mar)
	assert.True(t, marginAcc.Balance.IsZero())

}

func testFeeTransferContinuousNoFunds(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()

	trader := "mytrader"
	// create trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, err := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	require.NoError(t, err)

	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	require.NoError(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "mytrader",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
				MinAmount: num.NewUint(1000),
			},
		},
		tfa: map[string]uint64{trader: 1000},
	}

	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFeesReq)
	assert.Nil(t, transfers)
	assert.EqualError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())

}

func testFeeTransferContinuousNotEnoughFunds(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()
	trader := "mytrader"
	// create trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	general, err := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	require.NoError(t, err)

	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(context.Background(), general, num.NewUint(100))
	assert.Nil(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "mytrader",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
				MinAmount: num.NewUint(1000),
			},
		},
		tfa: map[string]uint64{trader: 1000},
	}

	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFeesReq)
	assert.Nil(t, transfers)
	assert.EqualError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())
}

func testFeeTransferContinuousOKWithEnoughInGenral(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()
	trader := "mytrader"
	// create trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	general, err := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	require.NoError(t, err)

	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(context.Background(), general, num.NewUint(10000))
	assert.Nil(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "mytrader",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
				MinAmount: num.NewUint(1000),
			},
		},
		tfa: map[string]uint64{trader: 1000},
	}

	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFeesReq)
	assert.NotNil(t, transfers)
	assert.NoError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())
	assert.Len(t, transfers, 1)
}

func testFeeTransferContinuousOKWith0Amount(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()
	trader := "mytrader"
	// create trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	general, err := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	require.NoError(t, err)

	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(context.Background(), general, num.NewUint(10000))
	assert.Nil(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "mytrader",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(0),
				},
				Type:      types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
				MinAmount: num.NewUint(0),
			},
		},
		tfa: map[string]uint64{trader: 1000},
	}

	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFeesReq)
	assert.NotNil(t, transfers)
	assert.NoError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())
	assert.Len(t, transfers, 1)
	generalAcc, _ := eng.GetAccountByID(general)
	assert.Equal(t, num.NewUint(10000), generalAcc.Balance)
}

func testFeeTransferContinuousOKWithEnoughInMargin(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()
	trader := "mytrader"
	// create trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, err := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	require.NoError(t, err)

	margin, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(context.Background(), margin, num.NewUint(10000))
	assert.Nil(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "mytrader",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
				MinAmount: num.NewUint(1000),
			},
		},
		tfa: map[string]uint64{trader: 1000},
	}

	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFeesReq)
	assert.NotNil(t, transfers)
	assert.NoError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())
	assert.Len(t, transfers, 1)
}

func testFeeTransferContinuousOKCheckAccountEvents(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()
	trader := "mytrader"
	// create trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, err := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	require.NoError(t, err)

	margin, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(context.Background(), margin, num.NewUint(10000))
	assert.Nil(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "mytrader",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
				MinAmount: num.NewUint(1000),
			},
			{
				Owner: "mytrader",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(3000),
				},
				Type:      types.TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY,
				MinAmount: num.NewUint(3000),
			},
		},
		tfa: map[string]uint64{trader: 1000},
	}

	var (
		seenLiqui bool
		seenInfra bool
	)
	eng.broker.EXPECT().Send(gomock.Any()).Times(4).Do(func(evt events.Event) {
		if evt.Type() != events.AccountEvent {
			t.FailNow()
		}
		accRaw := evt.(*events.Acc)
		acc := accRaw.Account()
		if acc.Type == types.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE {
			assert.Equal(t, 1000, int(acc.Balance))
			seenInfra = true
		}
		if acc.Type == types.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY {
			assert.Equal(t, 3000, int(acc.Balance))
			seenLiqui = true
		}
	})
	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFeesReq)
	assert.NotNil(t, transfers)
	assert.NoError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())
	assert.Len(t, transfers, 2)
	assert.True(t, seenInfra)
	assert.True(t, seenLiqui)
}

func testFeeTransferContinuousOKWithEnoughInGeneralAndMargin(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()
	trader := "mytrader"
	// create trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	general, err := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	require.NoError(t, err)

	margin, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	err = eng.Engine.UpdateBalance(context.Background(), general, num.NewUint(700))
	require.NoError(t, err)

	err = eng.Engine.UpdateBalance(context.Background(), margin, num.NewUint(900))
	require.NoError(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "mytrader",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY,
				MinAmount: num.NewUint(1000),
			},
		},
		tfa: map[string]uint64{trader: 1000},
	}

	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFeesReq)
	assert.NotNil(t, transfers)
	assert.NoError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())
	assert.Len(t, transfers, 1)

	// now check the balances
	// general should be empty
	generalAcc, _ := eng.GetAccountByID(general)
	assert.True(t, generalAcc.Balance.IsZero())
	marginAcc, _ := eng.GetAccountByID(margin)
	assert.Equal(t, num.NewUint(600), marginAcc.Balance)
}

func testEnableAssetSuccess(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()
	asset := types.Asset{
		Id: "MYASSET",
		Details: &types.AssetDetails{
			Symbol: "MYASSET",
		},
	}
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	err := eng.EnableAsset(context.Background(), asset)
	assert.NoError(t, err)
}

func testEnableAssetFailureDuplicate(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()
	asset := types.Asset{
		Id: "MYASSET",
		Details: &types.AssetDetails{
			Symbol: "MYASSET",
		},
	}
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	err := eng.EnableAsset(context.Background(), asset)
	assert.NoError(t, err)

	// now try to enable it again
	err = eng.EnableAsset(context.Background(), asset)
	assert.EqualError(t, err, collateral.ErrAssetAlreadyEnabled.Error())
}

func testCreateNewAccountForBadAsset(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	defer eng.Finish()

	_, err := eng.Engine.CreatePartyGeneralAccount(context.Background(), "sometrader", "notanasset")
	assert.EqualError(t, err, collateral.ErrInvalidAssetID.Error())
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), "sometrader", testMarketID, "notanasset")
	assert.EqualError(t, err, collateral.ErrInvalidAssetID.Error())
	_, _, err = eng.Engine.CreateMarketAccounts(context.Background(), "somemarketid", "notanasset")
	assert.EqualError(t, err, collateral.ErrInvalidAssetID.Error())
}

func testNew(t *testing.T) {
	eng := getTestEngine(t, "test-market")
	eng.Finish()
}

func testAddMarginAccount(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "funkytrader"

	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	margin, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// test balance is 0 when created
	acc, err := eng.Engine.GetAccountByID(margin)
	assert.Nil(t, err)
	assert.True(t, acc.Balance.IsZero())
}

func testAddMarginAccountFail(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "funkytrader"

	// create trader
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Error(t, err, collateral.ErrNoGeneralAccountWhenCreateMarginAccount)

}

func testAddTrader(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "funkytrader"

	// create trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	general, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	margin, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(context.Background(), general, num.NewUint(100000))
	assert.Nil(t, err)

	expectedGeneralBalance := num.NewUint(100000)

	// check the amount on each account now
	acc, err := eng.Engine.GetAccountByID(margin)
	assert.Nil(t, err)
	assert.True(t, acc.Balance.IsZero())

	acc, err = eng.Engine.GetAccountByID(general)
	assert.Nil(t, err)
	assert.Equal(t, expectedGeneralBalance, acc.Balance)

}

func testTransferLoss(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(10)

	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Mul(price, num.NewUint(5)))
	assert.Nil(t, err)

	// create trader accounts, set balance for money trader
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.UpdateBalance(context.Background(), marginMoneyTrader, num.NewUint(100000))
	assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
	}

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance of settlement account should be 3 times price
	assert.Equal(t, num.Sum(price, price), num.Sum(resp.Balances[0].Balance, responses[1].Balances[0].Balance))
	// there should be 2 ledger moves
	assert.Equal(t, 1, len(resp.Transfers))
}

func testTransferComplexLoss(t *testing.T) {
	trader := "test-trader"
	half := num.NewUint(500)
	price := num.Sum(half, half)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	// 5x price
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.Sum(price, price, price, price, price))
	assert.Nil(t, err)

	// create trader accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	marginTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.IncrementBalance(context.Background(), marginTrader, half)
	assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Asset:  "BTC",
				Amount: price,
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
	}
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos)
	assert.Equal(t, 1, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance should equal price (only 1 call after all)
	assert.Equal(t, price, resp.Balances[0].Balance)
	// there should be 2 ledger moves, one from trader account, one from insurance acc
	assert.Equal(t, 2, len(resp.Transfers))
}

func testTransferLossMissingTraderAccounts(t *testing.T) {
	trader := "test-trader"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Asset:  "BTC",
				Amount: price,
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
	}
	resp, err := eng.FinalSettlement(context.Background(), testMarketID, pos)
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account does not exist:")
}

func testDistributeWin(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, price)
	assert.Nil(t, err)

	// set settlement account
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.IncrementBalance(context.Background(), eng.marketSettlementID, num.Sum(price, price))
	assert.Nil(t, err)

	// create trader accounts, add balance for money trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)
	// err = eng.Engine.IncrementBalance(context.Background(), marginMoneyTrader, price*5)
	// assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_WIN,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_WIN,
		},
	}

	brokerExpMoneyBalance := price
	eng.broker.EXPECT().Send(gomock.Any()).Times(4).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, price.Uint64(), acc.Balance)
		}
		if acc.Owner == moneyTrader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, brokerExpMoneyBalance.Uint64(), acc.Balance)
			brokerExpMoneyBalance = num.NewUint(0).Add(brokerExpMoneyBalance, price)
		}
	})
	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos)
	assert.Equal(t, 2, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance of settlement account should be 3 times price
	for _, bal := range resp.Balances {
		if bal.Account.Type == types.AccountType_ACCOUNT_TYPE_SETTLEMENT {
			assert.Zero(t, bal.Account.Balance)
		}
	}
	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 1, len(resp.Transfers))
}

func testProcessBoth(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := num.NewUint(1000)
	priceX3 := num.Sum(price, price, price)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, priceX3)
	assert.Nil(t, err)

	// create trader accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.IncrementBalance(context.Background(), marginMoneyTrader, num.Sum(priceX3, price, price))
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_WIN,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_WIN,
		},
	}

	// next up, updating the balance of the traders' general accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(8).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == moneyTrader && acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			assert.Equal(t, int64(2000), acc.Balance)
		}
	})
	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos)
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err)
	resp := responses[0]
	// total balance of settlement account should be 3 times price
	for _, bal := range resp.Balances {
		if bal.Account.Type == types.AccountType_ACCOUNT_TYPE_SETTLEMENT {
			assert.True(t, bal.Account.Balance.IsZero())
		}
	}
	// resp = responses[1]
	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 1, len(responses[1].Transfers))
}

func testSettleBalanceNotZero(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create trader accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(8)
	gID, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	mID, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	assert.NotEmpty(t, mID)
	assert.NotEmpty(t, gID)

	// create + add balance
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(context.Background(), marginMoneyTrader, num.NewUint(0).Mul(num.NewUint(6), price))
	assert.Nil(t, err)
	pos := []*types.Transfer{
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(0).Mul(price, num.NewUint(2)), // lost 2xprice, trader only won half
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
	}

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	transfers := eng.getTestMTMTransfer(pos)
	_, _, err = eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	// this should return an error
	assert.Error(t, err)
	assert.Equal(t, collateral.ErrSettlementBalanceNotZero, err)
}

func testProcessBothProRated(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create trader accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(8)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.IncrementBalance(context.Background(), marginMoneyTrader, num.NewUint(0).Mul(price, num.NewUint(5)))
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_WIN,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_WIN,
		},
	}

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos)
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err)

	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 1, len(responses[1].Transfers))
}

func testProcessBothProRatedMTM(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create trader accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(8)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.IncrementBalance(context.Background(), marginMoneyTrader, num.NewUint(0).Mul(price, num.NewUint(5)))
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
	}

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	// quickly get the interface mocked for this test
	transfers := getMTMTransfer(pos)
	responses, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err, "was error")
	assert.NotEmpty(t, raw)

	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 1, len(raw[1].Transfers))
}

func testRemoveDistressedBalance(t *testing.T) {
	trader := "test-trader"

	insBalance := num.NewUint(1000)
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, insBalance)
	assert.Nil(t, err)

	// create trader accounts (calls buf.Add twice), and add balance (calls it a third time)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	marginID, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// add balance to margin account for trader
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.IncrementBalance(context.Background(), marginID, num.NewUint(100))
	assert.Nil(t, err)

	// events:
	data := []events.MarketPosition{
		marketPositionFake{
			party: trader,
		},
	}
	eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Id == marginID {
			assert.Zero(t, acc.Balance)
		} else {
			// this doesn't happen yet
			assert.Equal(t, num.NewUint(0).Add(insBalance, num.NewUint(100)).Uint64(), acc.Balance)
		}
	})
	resp, err := eng.RemoveDistressed(context.Background(), data, testMarketID, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.Transfers))

	// check if account was deleted
	_, err = eng.GetAccountByID(marginID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account does not exist:")
}

func testRemoveDistressedNoBalance(t *testing.T) {
	trader := "test-trader"

	insBalance := num.NewUint(1000)
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, insBalance)
	assert.Nil(t, err)

	// create trader accounts (calls buf.Add twice), and add balance (calls it a third time)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	marginID, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// no balance on margin account, so we don't expect there to be any balance updates in the buffer either
	// set up calls expected to buffer: add the update of the balance, of system account (insurance) and one with the margin account set to 0
	data := []events.MarketPosition{
		marketPositionFake{
			party: trader,
		},
	}
	resp, err := eng.RemoveDistressed(context.Background(), data, testMarketID, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(resp.Transfers))

	// check if account was deleted
	_, err = eng.GetAccountByID(marginID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account does not exist:")
}

// most of this function is copied from the MarkToMarket test - we're using channels, sure
// but the flow should remain the same regardless
func testMTMSuccess(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create trader accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(8)
	gID, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	mID, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	assert.NotEmpty(t, mID)
	assert.NotEmpty(t, gID)

	// create + add balance
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), moneyTrader, testMarketAsset)
	marginMoneyTrader, err := eng.Engine.CreatePartyMarginAccount(context.Background(), moneyTrader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.Engine.UpdateBalance(context.Background(), marginMoneyTrader, num.NewUint(0).Mul(num.NewUint(5), price))
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
		{
			Owner: moneyTrader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			assert.Equal(t, acc.Balance, int64(833))
		}
		if acc.Owner == moneyTrader && acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			assert.Equal(t, acc.Balance, int64(1666))
		}
	})
	transfers := eng.getTestMTMTransfer(pos)
	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(raw))
	assert.NotEmpty(t, evts)
}

func TestInvalidMarketID(t *testing.T) {
	trader := "test-trader"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create trader accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	invalidMarketID := testMarketID + "invalid"
	evts, raw, err := eng.MarkToMarket(context.Background(), invalidMarketID, transfers, "BTC")
	assert.Error(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestEmptyTransfer(t *testing.T) {
	trader := "test-trader"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create trader accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(0),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestNoMarginAccount(t *testing.T) {
	trader := "test-trader"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create trader accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.Error(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestNoGeneralAccount(t *testing.T) {
	trader := "test-trader"
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.Error(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestMTMNoTransfers(t *testing.T) {
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	pos := []*types.Transfer{}
	transfers := eng.getTestMTMTransfer(pos)

	// Empty list of transfers
	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)

	// List with a single nil value
	mt := mtmFake{
		t:     nil,
		party: "test-trader",
	}
	transfers = append(transfers, mt)
	evts, raw, err = eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Equal(t, len(evts), 1)
}

func TestFinalSettlementNoTransfers(t *testing.T) {
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	pos := []*types.Transfer{}

	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(responses))
}

func TestFinalSettlementNoSystemAccounts(t *testing.T) {
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: "testTrader",
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
	}

	responses, err := eng.FinalSettlement(context.Background(), "invalidMarketID", pos)
	assert.Error(t, err)
	assert.Equal(t, 0, len(responses))
}

func TestFinalSettlementNotEnoughMargin(t *testing.T) {
	amount := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(amount, num.NewUint(2)))
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), "testTrader", testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), "testTrader", testMarketID, testMarketAsset)
	require.NoError(t, err)

	pos := []*types.Transfer{
		{
			Owner: "testTrader",
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(0).Mul(amount, num.NewUint(100)),
				Asset:  "BTC",
			},
			Type: types.TransferType_TRANSFER_TYPE_LOSS,
		},
	}

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(responses))
}

func TestGetPartyMarginNoAccounts(t *testing.T) {
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	marketPos := mtmFake{
		party: "test-trader",
	}

	margin, err := eng.GetPartyMargin(marketPos, "BTC", testMarketID)
	assert.Nil(t, margin)
	assert.Error(t, err)
}

func TestGetPartyMarginNoMarginAccounts(t *testing.T) {
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), "test-trader", testMarketAsset)

	marketPos := mtmFake{
		party: "test-trader",
	}

	margin, err := eng.GetPartyMargin(marketPos, "BTC", testMarketID)
	assert.Nil(t, margin)
	assert.Error(t, err)
}

func TestGetPartyMarginEmpty(t *testing.T) {
	price := num.NewUint(1000)

	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.Id, num.NewUint(0).Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), "test-trader", testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), "test-trader", testMarketID, testMarketAsset)
	require.NoError(t, err)

	marketPos := mtmFake{
		party: "test-trader",
	}

	margin, err := eng.GetPartyMargin(marketPos, "BTC", testMarketID)
	assert.NotNil(t, margin)
	assert.Equal(t, margin.MarginBalance(), num.NewUint(0))
	assert.Equal(t, margin.GeneralBalance(), num.NewUint(0))
	assert.NoError(t, err)
}

func TestMTMLossSocialization(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	lossTrader1 := "losstrader1"
	lossTrader2 := "losstrader2"
	winTrader1 := "wintrader1"
	winTrader2 := "wintrader2"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(18)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), lossTrader1, testMarketAsset)
	margin, err := eng.Engine.CreatePartyMarginAccount(context.Background(), lossTrader1, testMarketID, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), margin, num.NewUint(500))
	assert.Nil(t, err)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), lossTrader2, testMarketAsset)
	margin, err = eng.Engine.CreatePartyMarginAccount(context.Background(), lossTrader2, testMarketID, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), margin, num.NewUint(1100))
	assert.Nil(t, err)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), winTrader1, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), winTrader1, testMarketID, testMarketAsset)
	// eng.Engine.IncrementBalance(context.Background(), margin, 0)
	assert.Nil(t, err)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), winTrader2, testMarketAsset)
	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), winTrader2, testMarketID, testMarketAsset)
	// eng.Engine.IncrementBalance(context.Background(), margin, 700)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: lossTrader1,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(700),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: lossTrader2,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(1400),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_LOSS,
		},
		{
			Owner: winTrader1,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(1400),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
		{
			Owner: winTrader2,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(700),
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_TRANSFER_TYPE_MTM_WIN,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == winTrader1 && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, uint64(1066), acc.Balance)
		}
		if acc.Owner == winTrader2 && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, uint64(534), acc.Balance)
		}
	})
	transfers := eng.getTestMTMTransfer(pos)
	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(raw))
	assert.NotEmpty(t, evts)
}

func testMarginUpdateOnOrderOK(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	acc, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100),
		transfer: &types.Transfer{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100),
			Type:      types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, acc.Balance, uint64(100))
		}
	})
	resp, closed, err := eng.Engine.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.Nil(t, err)
	assert.Nil(t, closed)
	assert.NotNil(t, resp)
}

func testMarginUpdateOnOrderOKNotShortFallWithBondAccount(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	acc, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), acc, num.NewUint(500))
	bondacc, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), bondacc, num.NewUint(500))
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100),
		transfer: &types.Transfer{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100),
			Type:      types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, acc.Balance, uint64(100))
		}
	})
	resp, closed, err := eng.Engine.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.Nil(t, err)
	assert.Nil(t, closed)
	assert.NotNil(t, resp)
}

func testMarginUpdateOnOrderOKUseBondAccount(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	genaccID, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), genaccID, num.NewUint(0))
	bondAccID, _ := eng.Engine.CreatePartyBondAccount(context.Background(), trader, testMarketID, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), bondAccID, num.NewUint(500))
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100),
		transfer: &types.Transfer{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100),
			Type:      types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, acc.Balance, uint64(100))
		}
	})
	resp, closed, err := eng.Engine.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.Nil(t, err)
	assert.NotNil(t, closed)
	assert.NotNil(t, resp)

	assert.Equal(t, closed.MarginShortFall(), num.NewUint(100))

	gacc, err := eng.Engine.GetAccountByID(genaccID)
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(0), gacc.Balance)
	bondAcc, err := eng.Engine.GetAccountByID(bondAccID)
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(400), bondAcc.Balance)
}

func testMarginUpdateOnOrderOKUseBondAndGeneralAccounts(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	genaccID, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), genaccID, num.NewUint(70))
	bondAccID, _ := eng.Engine.CreatePartyBondAccount(context.Background(), trader, testMarketID, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), bondAccID, num.NewUint(500))
	marginAccID, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100),
		transfer: &types.Transfer{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100),
			Type:      types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		// first call is to be updated to 70 with bond accoutns funds
		// then to 100 with general account funds
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.True(t, int(acc.Balance) == 70 || int(acc.Balance) == 100)
		}
	})

	resp, closed, err := eng.Engine.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.Nil(t, err)
	assert.NotNil(t, closed)
	assert.NotNil(t, resp)

	// we toped up only 70 in the bond account
	// but required 100 so we should pick 30 in the general account as well.

	// check shortfall
	assert.Equal(t, closed.MarginShortFall(), num.NewUint(30))

	gacc, err := eng.Engine.GetAccountByID(genaccID)
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(0), gacc.Balance)
	bondAcc, err := eng.Engine.GetAccountByID(bondAccID)
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(470), bondAcc.Balance)
	marginAcc, err := eng.Engine.GetAccountByID(marginAccID)
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(100), marginAcc.Balance)
}

func testMarginUpdateOnOrderOKThenRollback(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	acc, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100),
		transfer: &types.Transfer{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100),
			Type:      types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
		},
	}

	eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, int(acc.Balance), 100)
		}
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			assert.Equal(t, int(acc.Balance), 400)
		}
	})
	resp, closed, err := eng.Engine.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.Nil(t, err)
	assert.Nil(t, closed)
	assert.NotNil(t, resp)

	// then rollback
	rollback := &types.Transfer{
		Owner: trader,
		Amount: &types.FinancialAmount{
			Amount: num.NewUint(100),
			Asset:  testMarketAsset,
		},
		MinAmount: num.NewUint(100),
		Type:      types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
	}

	eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			assert.Equal(t, int(acc.Balance), 0)
		}
		if acc.Owner == trader && acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			assert.Equal(t, int(acc.Balance), 500)
		}
	})
	resp, err = eng.Engine.RollbackMarginUpdateOnOrder(context.Background(), testMarketID, testMarketAsset, rollback)
	assert.Nil(t, err)
	assert.NotNil(t, resp)

}

func testMarginUpdateOnOrderFail(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100000),
		transfer: &types.Transfer{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100000),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100000),
			Type:      types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
		},
	}

	resp, closed, err := eng.Engine.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.NotNil(t, err)
	assert.Error(t, err, collateral.ErrMinAmountNotReached.Error())
	assert.NotNil(t, closed)
	assert.Nil(t, resp)
}

func TestMarginUpdates(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	acc, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	list := make([]events.Risk, 1)

	list[0] = riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100),
		transfer: &types.Transfer{
			Owner: trader,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100),
			Type:      types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
		},
	}

	resp, margin, _, err := eng.Engine.MarginUpdate(context.Background(), testMarketID, list)
	assert.Nil(t, err)
	assert.Equal(t, len(margin), 0)
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Transfers[0].Amount, num.NewUint(100))
}

func TestClearMarket(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	acc, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	parties := []string{trader}

	responses, err := eng.Engine.ClearMarket(context.Background(), testMarketID, testMarketAsset, parties)

	assert.Nil(t, err)
	assert.Equal(t, len(responses), 1)
}

func TestClearMarketNoMargin(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	acc, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), acc, num.NewUint(500))

	parties := []string{trader}

	responses, err := eng.Engine.ClearMarket(context.Background(), testMarketID, testMarketAsset, parties)

	assert.NoError(t, err)
	assert.Equal(t, len(responses), 0)
}

func TestWithdrawalOK(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	acc, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	call := 0
	eng.broker.EXPECT().Send(gomock.Any()).Times(3).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			assert.Equal(t, 400, int(acc.Balance))
		} else if acc.Type == types.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW {
			// once to create the lock account, once to set its balance to 100
			assert.Equal(t, 100*call, int(acc.Balance))
			call++
		} else {
			t.FailNow()
		}
	})

	_, err = eng.Engine.LockFundsForWithdraw(context.Background(), trader, testMarketAsset, num.NewUint(100))
	assert.NoError(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Type == types.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW {
			assert.Equal(t, 0, int(acc.Balance))
		} else {
			t.FailNow()
		}
	})

	_, err = eng.Engine.Withdraw(context.Background(), trader, testMarketAsset, num.NewUint(100))
	assert.Nil(t, err)
}

func TestWithdrawalExact(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(5)
	acc, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		// fmt.Printf("ACCOUNT: %v\n", acc)
		if acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			assert.Equal(t, 0, int(acc.Balance))
		} else if acc.Type == types.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW {
			assert.Equal(t, 500, int(acc.Balance))
		} else {
			t.FailNow()
		}
	})

	_, err = eng.Engine.LockFundsForWithdraw(context.Background(), trader, testMarketAsset, num.NewUint(500))
	assert.NoError(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Type == types.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW {
			assert.Equal(t, 0, int(acc.Balance))
		} else {
			t.FailNow()
		}
	})

	_, err = eng.Engine.Withdraw(context.Background(), trader, testMarketAsset, num.NewUint(500))
	assert.Nil(t, err)
}

func TestWithdrawalNotEnough(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	acc, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_, err = eng.Engine.LockFundsForWithdraw(context.Background(), trader, testMarketAsset, num.NewUint(600))
	assert.EqualError(t, err, collateral.ErrNotEnoughFundsToWithdraw.Error())

}

func TestWithdrawalInvalidAccount(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	// create traders
	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	acc, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)
	eng.Engine.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.Engine.CreatePartyMarginAccount(context.Background(), trader, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_, err = eng.Engine.LockFundsForWithdraw(context.Background(), "invalid", testMarketAsset, num.NewUint(600))
	assert.Error(t, err)
	err = nil
	_, err = eng.Engine.Withdraw(context.Background(), "invalid", testMarketAsset, num.NewUint(600))
	assert.Error(t, err)
}

func TestChangeBalance(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()
	trader := "oktrader"

	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	acc, _ := eng.Engine.CreatePartyGeneralAccount(context.Background(), trader, testMarketAsset)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	eng.Engine.IncrementBalance(context.Background(), acc, num.NewUint(500))
	account, err := eng.Engine.GetAccountByID(acc)
	assert.NoError(t, err)
	assert.Equal(t, account.Balance, num.NewUint(500))

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	eng.Engine.IncrementBalance(context.Background(), acc, num.NewUint(250))
	account, err = eng.Engine.GetAccountByID(acc)
	require.NoError(t, err)
	assert.Equal(t, account.Balance, num.NewUint(750))

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	eng.Engine.UpdateBalance(context.Background(), acc, num.NewUint(666))
	account, err = eng.Engine.GetAccountByID(acc)
	require.NoError(t, err)
	assert.Equal(t, account.Balance, num.NewUint(666))

	err = eng.Engine.IncrementBalance(context.Background(), "invalid", num.NewUint(200))
	assert.Error(t, err, collateral.ErrAccountDoesNotExist)

	err = eng.Engine.UpdateBalance(context.Background(), "invalid", num.NewUint(300))
	assert.Error(t, err, collateral.ErrAccountDoesNotExist)
}

func TestReloadConfig(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	// Check that the log level is currently `debug`
	assert.Equal(t, eng.Engine.Level.Level, logging.DebugLevel)

	// Create a new config and make some changes to it
	newConfig := collateral.NewDefaultConfig()
	newConfig.Level = encoding.LogLevel{
		Level: logging.InfoLevel,
	}
	eng.Engine.ReloadConf(newConfig)

	// Verify that the log level has been changed
	assert.Equal(t, eng.Engine.Level.Level, logging.InfoLevel)
}

func (e *testEngine) getTestMTMTransfer(transfers []*types.Transfer) []events.Transfer {
	tt := make([]events.Transfer, 0, len(transfers))
	for _, t := range transfers {
		// Apply some limited validation here so we can filter out bad transfers
		if !t.Amount.Amount.IsZero() {
			mt := mtmFake{
				t:     t,
				party: t.Owner,
			}
			tt = append(tt, mt)
		}
	}
	return tt
}

func getTestEngine(t *testing.T, market string) *testEngine {
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	conf := collateral.NewDefaultConfig()
	conf.Level = encoding.LogLevel{Level: logging.DebugLevel}
	// 4 new events expected:
	// 2 markets accounts
	// 2 new assets
	broker.EXPECT().Send(gomock.Any()).Times(10)
	// system accounts created

	eng, err := collateral.New(logging.NewTestLogger(), conf, broker, time.Now())
	assert.Nil(t, err)

	// add the token asset
	tokAsset := types.Asset{
		Id: "VOTE",
		Details: &types.AssetDetails{
			Name:        "VOTE",
			Symbol:      "VOTE",
			Decimals:    5,
			TotalSupply: num.NewUint(1000),
			Source: &types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{},
			},
		},
	}
	err = eng.EnableAsset(context.Background(), tokAsset)
	assert.NoError(t, err)

	// enable the assert for the tests
	asset := types.Asset{
		Id: testMarketAsset,
		Details: &types.AssetDetails{
			Symbol: testMarketAsset,
		},
	}
	err = eng.EnableAsset(context.Background(), asset)
	assert.NoError(t, err)
	// ETH is added hardcoded in some places
	asset = types.Asset{
		Id: "ETH",
		Details: &types.AssetDetails{
			Symbol: "ETH",
		},
	}
	err = eng.EnableAsset(context.Background(), asset)
	assert.NoError(t, err)

	// create market and traders used for tests
	insID, setID, err := eng.CreateMarketAccounts(context.Background(), testMarketID, testMarketAsset)
	assert.Nil(t, err)

	return &testEngine{
		Engine:             eng,
		ctrl:               ctrl,
		broker:             broker,
		marketInsuranceID:  insID,
		marketSettlementID: setID,
		// systemAccs: accounts,
	}
}

func (e *testEngine) Finish() {
	e.systemAccs = nil
	e.ctrl.Finish()
}

type marketPositionFake struct {
	party           string
	size, buy, sell int64
	price           *num.Uint
	vwBuy, vwSell   *num.Uint
}

func (m marketPositionFake) Party() string     { return m.party }
func (m marketPositionFake) Size() int64       { return m.size }
func (m marketPositionFake) Buy() int64        { return m.buy }
func (m marketPositionFake) Sell() int64       { return m.sell }
func (m marketPositionFake) Price() *num.Uint  { return m.price }
func (m marketPositionFake) VWBuy() *num.Uint  { return m.vwBuy }
func (m marketPositionFake) VWSell() *num.Uint { return m.vwSell }
func (m marketPositionFake) ClearPotentials()  {}

type mtmFake struct {
	t     *types.Transfer
	party string
}

func (m mtmFake) Party() string             { return m.party }
func (m mtmFake) Size() int64               { return 0 }
func (m mtmFake) Price() *num.Uint          { return num.NewUint(0) }
func (m mtmFake) VWBuy() *num.Uint          { return num.NewUint(0) }
func (m mtmFake) VWSell() *num.Uint         { return num.NewUint(0) }
func (m mtmFake) Buy() int64                { return 0 }
func (m mtmFake) Sell() int64               { return 0 }
func (m mtmFake) ClearPotentials()          {}
func (m mtmFake) Transfer() *types.Transfer { return m.t }

func getMTMTransfer(transfers []*types.Transfer) []events.Transfer {
	r := make([]events.Transfer, 0, len(transfers))
	for _, t := range transfers {
		r = append(r, &mtmFake{
			t:     t,
			party: t.Owner,
		})
	}
	return r
}

type riskFake struct {
	party           string
	size, buy, sell int64
	price           *num.Uint
	vwBuy, vwSell   *num.Uint
	margins         *types.MarginLevels
	amount          *num.Uint
	transfer        *types.Transfer
	asset           string
	marginShortFall *num.Uint
}

func (m riskFake) Party() string                     { return m.party }
func (m riskFake) Size() int64                       { return m.size }
func (m riskFake) Buy() int64                        { return m.buy }
func (m riskFake) Sell() int64                       { return m.sell }
func (m riskFake) Price() *num.Uint                  { return m.price }
func (m riskFake) VWBuy() *num.Uint                  { return m.vwBuy }
func (m riskFake) VWSell() *num.Uint                 { return m.vwSell }
func (m riskFake) ClearPotentials()                  {}
func (m riskFake) Transfer() *types.Transfer         { return m.transfer }
func (m riskFake) Amount() *num.Uint                 { return m.amount }
func (m riskFake) MarginLevels() *types.MarginLevels { return m.margins }
func (m riskFake) Asset() string                     { return m.asset }
func (m riskFake) MarketID() string                  { return "" }
func (m riskFake) MarginBalance() *num.Uint          { return num.NewUint(0) }
func (m riskFake) GeneralBalance() *num.Uint         { return num.NewUint(0) }
func (m riskFake) BondBalance() *num.Uint            { return num.NewUint(0) }
func (m riskFake) MarginShortFall() *num.Uint        { return m.marginShortFall }

type transferFees struct {
	tfs []*types.Transfer
	tfa map[string]uint64
}

func (t transferFees) Transfers() []*types.Transfer { return t.tfs }

func (t transferFees) TotalFeesAmountPerParty() map[string]*num.Uint {
	ret := make(map[string]*num.Uint, len(t.tfa)) // convert in here, so the tests are easier to read
	for k, v := range t.tfa {
		ret[k] = num.NewUint(v)
	}
	return ret
}

func TestHash(t *testing.T) {
	eng := getTestEngine(t, testMarketID)
	defer eng.Finish()

	// Create the accounts
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	id1, err := eng.Engine.CreatePartyGeneralAccount(context.Background(), "t1", testMarketAsset)
	require.NoError(t, err)

	id2, err := eng.Engine.CreatePartyGeneralAccount(context.Background(), "t2", testMarketAsset)
	require.NoError(t, err)

	_, err = eng.Engine.CreatePartyMarginAccount(context.Background(), "t1", testMarketID, testMarketAsset)
	require.NoError(t, err)

	// Add balances
	require.NoError(t,
		eng.Engine.UpdateBalance(context.Background(), id1, num.NewUint(100)),
	)

	require.NoError(t,
		eng.Engine.UpdateBalance(context.Background(), id2, num.NewUint(500)),
	)

	hash := eng.Hash()
	require.Equal(t,
		"e503fd3b86b7d16599c7db430c7d4e229d46ae65e92de319fb7554a62532ae75",
		hex.EncodeToString(hash),
		"It should match against the known hash",
	)
	// compute the hash 100 times for determinism verification
	for i := 0; i < 100; i++ {
		got := eng.Hash()
		require.Equal(t, hash, got)
	}

}
