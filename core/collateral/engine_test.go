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

package collateral_test

import (
	"context"
	"encoding/hex"
	"strconv"
	"testing"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/collateral/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	ptypes "code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testMarketID    = "7CPSHJB35AIQBTNMIE6NLFPZGHOYRQ3D"
	testMarketAsset = "BTC"
	rewardsID       = "0000000000000000000000000000000000000000000000000000000000000000"
)

type testEngine struct {
	*collateral.Engine
	ctrl               *gomock.Controller
	timeSvc            *mocks.MockTimeService
	broker             *bmocks.MockBroker
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
	t.Run("test collecting buys - both insurance and sufficient in party accounts", testTransferLoss)
	t.Run("test collecting buys - party account not empty, but insufficient", testTransferComplexLoss)
	t.Run("test collecting buys - party missing some accounts", testTransferLossMissingPartyAccounts)
	t.Run("test collecting both buys and sells - Successfully collect buy and sell in a single call", testProcessBoth)
	t.Run("test distribution insufficient funds - Transfer losses (partial), distribute wins pro-rate", testProcessBothProRated)
	t.Run("test releas party margin account", testReleasePartyMarginAccount)
}

func TestCollateralMarkToMarket(t *testing.T) {
	t.Run("Mark to Market distribution, insufficient funds - complex scenario", testProcessBothProRatedMTM)
	t.Run("Mark to Market successful", testMTMSuccess)
	// we panic if settlement account is non-zero, this test doesn't pass anymore
	t.Run("Mark to Market wins and losses do not match up, settlement not drained", testSettleBalanceNotZero)
}

func TestAddPartyToMarket(t *testing.T) {
	t.Run("Successful calls adding new parties (one duplicate, one actual new)", testAddParty)
	t.Run("Can add a party margin account if general account for asset exists", testAddMarginAccount)
	t.Run("Fail add party margin account if no general account for asset exisrts", testAddMarginAccountFail)
}

func TestRemoveDistressed(t *testing.T) {
	t.Run("Successfully remove distressed party and transfer balance", testRemoveDistressedBalance)
	t.Run("Successfully remove distressed party, no balance transfer", testRemoveDistressedNoBalance)
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
	t.Run("test a party with an account has a balance", testPartyWithAccountHasABalance)
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

func TestTransferRewards(t *testing.T) {
	t.Run("transfer rewards empty slice", testTransferRewardsEmptySlice)
	t.Run("transfer rewards missing rewards account", testTransferRewardsNoRewardsAccount)
	t.Run("transfer rewards success", testTransferRewardsSuccess)
}

func TestClearAccounts(t *testing.T) {
	t.Run("clear fee accounts", testClearFeeAccounts)
}

func testClearFeeAccounts(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	ctx := context.Background()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	mktID := "market"
	asset := "ETH"
	party := "myparty"
	assetT := types.Asset{
		ID: asset,
		Details: &types.AssetDetails{
			Symbol: asset,
		},
	}

	eng.EnableAsset(ctx, assetT)
	_, _ = eng.GetGlobalRewardAccount(asset)
	_, _, err := eng.CreateMarketAccounts(ctx, mktID, asset)
	require.NoError(t, err)
	general, err := eng.CreatePartyGeneralAccount(ctx, party, asset)
	require.NoError(t, err)

	_, err = eng.CreatePartyMarginAccount(ctx, party, mktID, asset)
	require.NoError(t, err)

	// add funds
	err = eng.UpdateBalance(ctx, general, num.NewUint(10000))
	assert.Nil(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: party,
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferTypeMakerFeePay,
				MinAmount: num.NewUint(1000),
			},
		},
		tfa: map[string]uint64{party: 1000},
	}

	transfers, err := eng.TransferFeesContinuousTrading(ctx, mktID, asset, transferFeesReq)
	assert.NotNil(t, transfers)
	assert.NoError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())
	assert.Len(t, transfers, 1)

	assert.Equal(t, 1, len(transfers[0].Entries))
	assert.Equal(t, num.NewUint(9000), transfers[0].Entries[0].FromAccountBalance)
	assert.Equal(t, num.NewUint(1000), transfers[0].Entries[0].ToAccountBalance)
	eng.ClearInsurancepool(ctx, mktID, asset, true)
}

func testTransferRewardsEmptySlice(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	res, err := eng.TransferRewards(context.Background(), "reward", []*types.Transfer{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(res))
}

func testTransferRewardsNoRewardsAccount(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	transfers := []*types.Transfer{
		{
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(1000),
				Asset:  "ETH",
			},
			MinAmount: num.NewUint(1000),
			Type:      types.TransferTypeRewardPayout,
			Owner:     "party1",
		},
	}

	res, err := eng.TransferRewards(context.Background(), "rewardAccID", transfers)
	require.Error(t, errors.New("account does not exists"), err)
	require.Nil(t, res)
}

func testTransferRewardsSuccess(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	rewardAcc, _ := eng.GetGlobalRewardAccount("ETH")

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	eng.IncrementBalance(context.Background(), rewardAcc.ID, num.NewUint(1000))

	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	partyAccountID, _ := eng.CreatePartyGeneralAccount(context.Background(), "party1", "ETH")

	transfers := []*types.Transfer{
		{
			Owner: "party1",
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(1000),
				Asset:  "ETH",
			},
			MinAmount: num.NewUint(1000),
			Type:      types.TransferTypeRewardPayout,
		},
	}

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	lm, err := eng.TransferRewards(context.Background(), rewardAcc.ID, transfers)
	require.Nil(t, err)
	partyAccount, _ := eng.GetAccountByID(partyAccountID)
	require.Equal(t, num.NewUint(1000), partyAccount.Balance)

	rewardAccount, _ := eng.GetGlobalRewardAccount("ETH")
	require.Equal(t, num.UintZero(), rewardAccount.Balance)

	assert.Equal(t, 1, len(lm))
	assert.Equal(t, 1, len(lm[0].Entries))
	assert.Equal(t, num.NewUint(1000), lm[0].Entries[0].ToAccountBalance)
	assert.Equal(t, num.UintZero(), lm[0].Entries[0].FromAccountBalance)
}

func testPartyWithAccountHasABalance(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	party := "myparty"
	bal := num.NewUint(500)
	// create party
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	acc, err := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	assert.NoError(t, err)

	// then add some money
	err = eng.UpdateBalance(context.Background(), acc, bal)
	assert.Nil(t, err)

	evt := eng.broker.GetLastByTypeAndID(events.AccountEvent, acc)
	require.NotNil(t, evt)
	_, ok := evt.(accEvt)
	require.True(t, ok)
}

func testCreateBondAccountFailureNoGeneral(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	party := "myparty"
	// create party
	_, err := eng.CreatePartyBondAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.EqualError(t, err, "party general account missing when trying to create a bond account")
}

func testCreateBondAccountSuccess(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	party := "myparty"
	bal := num.NewUint(500)
	// create party
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, err := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	require.NoError(t, err)
	bnd, err := eng.CreatePartyBondAccount(context.Background(), party, testMarketID, testMarketAsset)
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
	require.Equal(t, bal.String(), account.Balance)
	// these two checks are a bit redundant at this point
	// but at least we're verifying that the GetAccountByID and the latest event return the same state
	bndacc, _ := eng.GetAccountByID(bnd)
	assert.Equal(t, account.Balance, bndacc.Balance.String())
}

func TestDeleteBondAccount(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)

	party := "myparty"
	err := eng.RemoveBondAccount(party, testMarketID, testMarketAsset)
	require.EqualError(t, err, collateral.ErrAccountDoesNotExist.Error())

	bal := num.NewUint(500)
	// create party
	_, err = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	require.NoError(t, err)
	bnd, err := eng.CreatePartyBondAccount(context.Background(), party, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	err = eng.UpdateBalance(context.Background(), bnd, bal)
	require.Nil(t, err)

	require.Panics(t, func() { eng.RemoveBondAccount(party, testMarketID, testMarketAsset) })

	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: bal,
			Asset:  testMarketAsset,
		},
		Type:      types.TransferTypeBondHigh,
		MinAmount: bal,
	}

	_, err = eng.BondUpdate(context.Background(), testMarketID, transfer)
	require.NoError(t, err)

	err = eng.RemoveBondAccount(party, testMarketID, testMarketAsset)
	require.NoError(t, err)

	_, err = eng.GetPartyBondAccount(testMarketID, party, testMarketAsset)

	require.ErrorContains(t, err, "account does not exist")
}

func testFeesTransferContinuousNoTransfer(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFees{})
	assert.Nil(t, transfers)
	assert.Nil(t, err)
}

func testReleasePartyMarginAccount(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	party := "myparty"
	// create party
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	gen, err := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	require.NoError(t, err)

	mar, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.UpdateBalance(context.Background(), gen, num.NewUint(100))
	assert.Nil(t, err)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.UpdateBalance(context.Background(), mar, num.NewUint(500))
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	lm, err := eng.ClearPartyMarginAccount(
		context.Background(), party, testMarketID, testMarketAsset)
	assert.NoError(t, err)
	generalAcc, _ := eng.GetAccountByID(gen)
	assert.Equal(t, num.NewUint(600), generalAcc.Balance)
	marginAcc, _ := eng.GetAccountByID(mar)
	assert.True(t, marginAcc.Balance.IsZero())

	assert.Equal(t, 1, len(lm.Entries))
	assert.Equal(t, num.NewUint(600), lm.Entries[0].ToAccountBalance)
	assert.Equal(t, num.UintZero(), lm.Entries[0].FromAccountBalance)
}

func testFeeTransferContinuousNoFunds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	party := "myparty"
	// create party
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, err := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	require.NoError(t, err)

	_, err = eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	require.NoError(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "myparty",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferTypeInfrastructureFeePay,
				MinAmount: num.NewUint(1000),
			},
		},
		tfa: map[string]uint64{party: 1000},
	}

	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFeesReq)
	assert.Nil(t, transfers)
	assert.EqualError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())
}

func testFeeTransferContinuousNotEnoughFunds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "myparty"
	// create party
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	general, err := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	require.NoError(t, err)

	_, err = eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.UpdateBalance(context.Background(), general, num.NewUint(100))
	assert.Nil(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "myparty",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferTypeInfrastructureFeePay,
				MinAmount: num.NewUint(1000),
			},
		},
		tfa: map[string]uint64{party: 1000},
	}

	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFeesReq)
	assert.Nil(t, transfers)
	assert.EqualError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())
}

func testFeeTransferContinuousOKWithEnoughInGenral(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "myparty"
	// create party
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	general, err := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	require.NoError(t, err)

	_, err = eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.UpdateBalance(context.Background(), general, num.NewUint(10000))
	assert.Nil(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "myparty",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferTypeInfrastructureFeePay,
				MinAmount: num.NewUint(1000),
			},
		},
		tfa: map[string]uint64{party: 1000},
	}

	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFeesReq)
	assert.NotNil(t, transfers)
	assert.NoError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())
	assert.Len(t, transfers, 1)

	assert.Equal(t, 1, len(transfers[0].Entries))
	assert.Equal(t, num.NewUint(9000), transfers[0].Entries[0].FromAccountBalance)
	assert.Equal(t, num.NewUint(1000), transfers[0].Entries[0].ToAccountBalance)
}

func testFeeTransferContinuousOKWith0Amount(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "myparty"
	// create party
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	general, err := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	require.NoError(t, err)

	_, err = eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.UpdateBalance(context.Background(), general, num.NewUint(10000))
	assert.Nil(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "myparty",
				Amount: &types.FinancialAmount{
					Amount: num.UintZero(),
				},
				Type:      types.TransferTypeInfrastructureFeePay,
				MinAmount: num.UintZero(),
			},
		},
		tfa: map[string]uint64{party: 1000},
	}

	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFeesReq)
	assert.NotNil(t, transfers)
	assert.NoError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())
	assert.Len(t, transfers, 1)
	generalAcc, _ := eng.GetAccountByID(general)
	assert.Equal(t, num.NewUint(10000), generalAcc.Balance)

	assert.Len(t, transfers[0].Entries, 1)
	assert.Equal(t, num.UintZero(), transfers[0].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(10000), transfers[0].Entries[0].FromAccountBalance)
}

func testFeeTransferContinuousOKWithEnoughInMargin(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "myparty"
	// create party
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, err := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	require.NoError(t, err)

	margin, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.UpdateBalance(context.Background(), margin, num.NewUint(10000))
	assert.Nil(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "myparty",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferTypeInfrastructureFeePay,
				MinAmount: num.NewUint(1000),
			},
		},
		tfa: map[string]uint64{party: 1000},
	}

	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	transfers, err := eng.TransferFeesContinuousTrading(
		context.Background(), testMarketID, testMarketAsset, transferFeesReq)
	assert.NotNil(t, transfers)
	assert.NoError(t, err, collateral.ErrInsufficientFundsToPayFees.Error())
	assert.Len(t, transfers, 1)
	assert.Len(t, transfers[0].Entries, 1)
	assert.Equal(t, num.NewUint(1000), transfers[0].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(9000), transfers[0].Entries[0].FromAccountBalance)
}

func testFeeTransferContinuousOKCheckAccountEvents(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "myparty"
	// create party
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, err := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	require.NoError(t, err)

	margin, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.UpdateBalance(context.Background(), margin, num.NewUint(10000))
	assert.Nil(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "myparty",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferTypeInfrastructureFeePay,
				MinAmount: num.NewUint(1000),
			},
			{
				Owner: "myparty",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(3000),
				},
				Type:      types.TransferTypeLiquidityFeePay,
				MinAmount: num.NewUint(3000),
			},
		},
		tfa: map[string]uint64{party: 1000},
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
		if acc.Type == types.AccountTypeFeesInfrastructure {
			assert.Equal(t, 1000, stringToInt(acc.Balance))
			seenInfra = true
		}
		if acc.Type == types.AccountTypeFeesLiquidity {
			assert.Equal(t, 3000, stringToInt(acc.Balance))
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

	assert.Equal(t, num.NewUint(1000), transfers[0].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(9000), transfers[0].Entries[0].FromAccountBalance)
	assert.Equal(t, num.NewUint(3000), transfers[1].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(6000), transfers[1].Entries[0].FromAccountBalance)
}

func testFeeTransferContinuousOKWithEnoughInGeneralAndMargin(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "myparty"
	// create party
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	general, err := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	require.NoError(t, err)

	margin, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	require.NoError(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	err = eng.UpdateBalance(context.Background(), general, num.NewUint(700))
	require.NoError(t, err)

	err = eng.UpdateBalance(context.Background(), margin, num.NewUint(900))
	require.NoError(t, err)

	transferFeesReq := transferFees{
		tfs: []*types.Transfer{
			{
				Owner: "myparty",
				Amount: &types.FinancialAmount{
					Amount: num.NewUint(1000),
				},
				Type:      types.TransferTypeInfrastructureFeePay,
				MinAmount: num.NewUint(1000),
			},
		},
		tfa: map[string]uint64{party: 1000},
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
	assert.Equal(t, num.NewUint(700), transfers[0].Entries[0].ToAccountBalance)
	assert.Equal(t, num.UintZero(), transfers[0].Entries[0].FromAccountBalance)
}

func testEnableAssetSuccess(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	asset := types.Asset{
		ID: "MYASSET",
		Details: &types.AssetDetails{
			Symbol: "MYASSET",
		},
	}
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	err := eng.EnableAsset(context.Background(), asset)
	assert.NoError(t, err)

	assetInsuranceAcc, _ := eng.GetGlobalRewardAccount(asset.ID)
	assert.True(t, assetInsuranceAcc.Balance.IsZero())
}

func testEnableAssetFailureDuplicate(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	asset := types.Asset{
		ID: "MYASSET",
		Details: &types.AssetDetails{
			Symbol: "MYASSET",
		},
	}
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	err := eng.EnableAsset(context.Background(), asset)
	assert.NoError(t, err)

	// now try to enable it again
	err = eng.EnableAsset(context.Background(), asset)
	assert.EqualError(t, err, collateral.ErrAssetAlreadyEnabled.Error())
}

func testCreateNewAccountForBadAsset(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	_, err := eng.CreatePartyGeneralAccount(context.Background(), "someparty", "notanasset")
	assert.EqualError(t, err, collateral.ErrInvalidAssetID.Error())
	_, err = eng.CreatePartyMarginAccount(context.Background(), "someparty", testMarketID, "notanasset")
	assert.EqualError(t, err, collateral.ErrInvalidAssetID.Error())
	_, _, err = eng.CreateMarketAccounts(context.Background(), "somemarketid", "notanasset")
	assert.EqualError(t, err, collateral.ErrInvalidAssetID.Error())
}

func testNew(t *testing.T) {
	eng := getTestEngine(t)
	eng.Finish()
}

func testAddMarginAccount(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "funkyparty"

	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	margin, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// test balance is 0 when created
	acc, err := eng.GetAccountByID(margin)
	assert.Nil(t, err)
	assert.True(t, acc.Balance.IsZero())
}

func testAddMarginAccountFail(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "funkyparty"

	// create party
	_, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Error(t, err, collateral.ErrNoGeneralAccountWhenCreateMarginAccount)
}

func testAddParty(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "funkyparty"

	// create party
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	general, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	margin, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// add funds
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.UpdateBalance(context.Background(), general, num.NewUint(100000))
	assert.Nil(t, err)

	expectedGeneralBalance := num.NewUint(100000)

	// check the amount on each account now
	acc, err := eng.GetAccountByID(margin)
	assert.Nil(t, err)
	assert.True(t, acc.Balance.IsZero())

	acc, err = eng.GetAccountByID(general)
	assert.Nil(t, err)
	assert.Equal(t, expectedGeneralBalance, acc.Balance)
}

func testTransferLoss(t *testing.T) {
	party := "test-party"
	moneyParty := "money-party"

	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(9)

	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Mul(price, num.NewUint(5)))
	assert.Nil(t, err)

	// create party accounts, set balance for money party
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	_, err = eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), moneyParty, testMarketAsset)
	marginMoneyParty, err := eng.CreatePartyMarginAccount(context.Background(), moneyParty, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.UpdateBalance(context.Background(), marginMoneyParty, num.NewUint(100000))
	assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeLoss,
		},
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeWin,
		},
	}

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos, num.UintOne())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance of settlement account should be 2 times price
	assert.Equal(t, num.Sum(price, price), num.Sum(resp.Balances[0].Balance, responses[1].Balances[0].Balance))
	// there should be 1 ledger moves
	assert.Equal(t, 1, len(resp.Entries))
	assert.Equal(t, num.NewUint(4000), resp.Entries[0].FromAccountBalance)
	assert.Equal(t, num.NewUint(1000), resp.Entries[0].ToAccountBalance)

	assert.Equal(t, 1, len(responses[1].Entries))
	assert.Equal(t, num.UintZero(), responses[1].Entries[0].FromAccountBalance)
	assert.Equal(t, num.NewUint(101000), responses[1].Entries[0].ToAccountBalance)
}

func testTransferComplexLoss(t *testing.T) {
	party := "test-party"
	moneyParty := "money-party"
	half := num.NewUint(500)
	price := num.Sum(half, half)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(10)

	_, _ = eng.CreatePartyGeneralAccount(context.Background(), moneyParty, testMarketAsset)
	marginMoneyParty, err := eng.CreatePartyMarginAccount(context.Background(), moneyParty, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	err = eng.UpdateBalance(context.Background(), marginMoneyParty, num.NewUint(100000))
	assert.Nil(t, err)

	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	// 5x price
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.Sum(price, price, price, price, price))
	assert.Nil(t, err)

	// create party accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	marginParty, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.IncrementBalance(context.Background(), marginParty, half)
	assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Asset:  "BTC",
				Amount: price,
			},
			Type: types.TransferTypeLoss,
		},
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Asset:  "BTC",
				Amount: price,
			},
			Type: types.TransferTypeWin,
		},
	}

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos, num.UintOne())
	assert.Equal(t, 2, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance should equal price (only 1 call after all)
	assert.Equal(t, price, resp.Balances[0].Balance)
	// there should be 2 ledger moves, one from party account, one from insurance acc
	assert.Equal(t, 2, len(resp.Entries))

	assert.Equal(t, 2, len(responses[0].Entries))
	assert.Equal(t, num.UintZero(), responses[0].Entries[0].FromAccountBalance)
	assert.Equal(t, num.NewUint(500), responses[0].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(4500), responses[0].Entries[1].FromAccountBalance)
	assert.Equal(t, num.NewUint(1000), responses[0].Entries[1].ToAccountBalance)

	assert.Equal(t, 1, len(responses[1].Entries))
	assert.Equal(t, num.UintZero(), responses[1].Entries[0].FromAccountBalance)
	assert.Equal(t, num.NewUint(101000), responses[1].Entries[0].ToAccountBalance)
}

func testTransferLossMissingPartyAccounts(t *testing.T) {
	party := "test-party"
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Asset:  "BTC",
				Amount: price,
			},
			Type: types.TransferTypeLoss,
		},
	}
	resp, err := eng.FinalSettlement(context.Background(), testMarketID, pos, num.UintOne())
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account does not exist:")
}

func testProcessBoth(t *testing.T) {
	party := "test-party"
	moneyParty := "money-party"
	price := num.NewUint(1000)
	priceX3 := num.Sum(price, price, price)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, priceX3)
	assert.Nil(t, err)

	// create party accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	_, err = eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_, _ = eng.CreatePartyGeneralAccount(context.Background(), moneyParty, testMarketAsset)
	marginMoneyParty, err := eng.CreatePartyMarginAccount(context.Background(), moneyParty, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.IncrementBalance(context.Background(), marginMoneyParty, num.Sum(priceX3, price, price))
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeLoss,
		},
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeLoss,
		},
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeWin,
		},
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeWin,
		},
	}

	// next up, updating the balance of the parties' general accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(8).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == moneyParty && acc.Type == types.AccountTypeGeneral {
			assert.Equal(t, int64(2000), acc.Balance)
		}
	})
	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos, num.UintOne())
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err)
	resp := responses[0]
	// total balance of settlement account should be 3 times price
	for _, bal := range resp.Balances {
		if bal.Account.Type == types.AccountTypeSettlement {
			assert.True(t, bal.Account.Balance.IsZero())
		}
	}
	// resp = responses[1]
	// there should be 3 ledger moves -> settle to party 1, settle to party 2, insurance to party 2
	assert.Equal(t, 1, len(responses[0].Entries))
	for _, e := range responses[0].Entries {
		assert.Equal(t, num.NewUint(2000), e.FromAccountBalance)
		assert.Equal(t, num.NewUint(1000), e.ToAccountBalance)
	}

	assert.Equal(t, 1, len(responses[1].Entries))
	for _, e := range responses[1].Entries {
		assert.Equal(t, num.NewUint(4000), e.FromAccountBalance)
		assert.Equal(t, num.NewUint(2000), e.ToAccountBalance)
	}

	assert.Equal(t, 1, len(responses[2].Entries))
	for _, e := range responses[2].Entries {
		assert.Equal(t, num.NewUint(1000), e.FromAccountBalance)
		assert.Equal(t, num.NewUint(1000), e.ToAccountBalance)
	}

	assert.Equal(t, 1, len(responses[3].Entries))
	for _, e := range responses[3].Entries {
		assert.Equal(t, num.UintZero(), e.FromAccountBalance)
		assert.Equal(t, num.NewUint(5000), e.ToAccountBalance)
	}
}

func TestLossSocialization(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	lossParty1 := "lossparty1"
	lossParty2 := "lossparty2"
	winParty1 := "winparty1"
	winParty2 := "winparty2"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(18)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), lossParty1, testMarketAsset)
	margin, err := eng.CreatePartyMarginAccount(context.Background(), lossParty1, testMarketID, testMarketAsset)
	eng.IncrementBalance(context.Background(), margin, num.NewUint(500))
	assert.Nil(t, err)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), lossParty2, testMarketAsset)
	margin, err = eng.CreatePartyMarginAccount(context.Background(), lossParty2, testMarketID, testMarketAsset)
	eng.IncrementBalance(context.Background(), margin, num.NewUint(1100))
	assert.Nil(t, err)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), winParty1, testMarketAsset)
	_, err = eng.CreatePartyMarginAccount(context.Background(), winParty1, testMarketID, testMarketAsset)
	assert.Nil(t, err)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), winParty2, testMarketAsset)
	_, err = eng.CreatePartyMarginAccount(context.Background(), winParty2, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	transfers := []*types.Transfer{
		{
			Owner: lossParty1,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(700),
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeLoss,
		},
		{
			Owner: lossParty2,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(1400),
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeLoss,
		},
		{
			Owner: winParty1,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(1400),
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeWin,
		},
		{
			Owner: winParty2,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(700),
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeWin,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == winParty1 && acc.Type == types.AccountTypeMargin {
			assert.Equal(t, 1066, stringToInt(acc.Balance))
		}
		if acc.Owner == winParty2 && acc.Type == types.AccountTypeMargin {
			assert.Equal(t, 534, stringToInt(acc.Balance))
		}
	})
	raw, err := eng.FinalSettlement(context.Background(), testMarketID, transfers, num.UintOne())
	assert.NoError(t, err)
	assert.Equal(t, 4, len(raw))

	assert.Equal(t, 1, len(raw[0].Entries))
	assert.Equal(t, num.NewUint(500), raw[0].Entries[0].ToAccountBalance)
	assert.Equal(t, 1, len(raw[1].Entries))
	assert.Equal(t, num.NewUint(1600), raw[1].Entries[0].ToAccountBalance)
	assert.Equal(t, 1, len(raw[2].Entries))
	assert.Equal(t, num.NewUint(1066), raw[2].Entries[0].ToAccountBalance)
	assert.Equal(t, 1, len(raw[3].Entries))
	assert.Equal(t, num.NewUint(534), raw[3].Entries[0].ToAccountBalance)
}

func testSettleBalanceNotZero(t *testing.T) {
	party := "test-party"
	moneyParty := "money-party"
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create party accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(8)
	gID, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	mID, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	assert.NotEmpty(t, mID)
	assert.NotEmpty(t, gID)

	// create + add balance
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), moneyParty, testMarketAsset)
	marginMoneyParty, err := eng.CreatePartyMarginAccount(context.Background(), moneyParty, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.UpdateBalance(context.Background(), marginMoneyParty, num.UintZero().Mul(num.NewUint(6), price))
	assert.Nil(t, err)
	pos := []*types.Transfer{
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Amount: num.UintZero().Mul(price, num.NewUint(2)), // lost 2xprice, party only won half
				Asset:  "BTC",
			},
			Type: types.TransferTypeMTMLoss,
		},
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeMTMWin,
		},
	}

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	transfers := eng.getTestMTMTransfer(pos)
	defer func() {
		r := recover()
		require.NotNil(t, r)
	}()
	_, _, _ = eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	// this should return an error
}

func testProcessBothProRated(t *testing.T) {
	party := "test-party"
	moneyParty := "money-party"
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create party accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(8)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	_, err = eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_, _ = eng.CreatePartyGeneralAccount(context.Background(), moneyParty, testMarketAsset)
	marginMoneyParty, err := eng.CreatePartyMarginAccount(context.Background(), moneyParty, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.IncrementBalance(context.Background(), marginMoneyParty, num.UintZero().Mul(price, num.NewUint(5)))
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeLoss,
		},
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeLoss,
		},
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeWin,
		},
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeWin,
		},
	}

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(2)
	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos, num.UintOne())
	assert.Equal(t, 4, len(responses))
	assert.NoError(t, err)

	// there should be 3 ledger moves -> settle to party 1, settle to party 2, insurance to party 2
	assert.Equal(t, 1, len(responses[0].Entries))
	assert.Equal(t, num.NewUint(500), responses[0].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(500), responses[0].Entries[0].ToAccountBalance)

	assert.Equal(t, 1, len(responses[1].Entries))
	assert.Equal(t, num.NewUint(1500), responses[1].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(1500), responses[1].Entries[0].ToAccountBalance)

	assert.Equal(t, 1, len(responses[2].Entries))
	assert.Equal(t, num.NewUint(750), responses[2].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(750), responses[2].Entries[0].ToAccountBalance)

	assert.Equal(t, 1, len(responses[3].Entries))
	assert.Equal(t, num.NewUint(4750), responses[3].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(4750), responses[3].Entries[0].ToAccountBalance)
}

func testProcessBothProRatedMTM(t *testing.T) {
	party := "test-party"
	moneyParty := "money-party"
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create party accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(8)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	_, err = eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_, _ = eng.CreatePartyGeneralAccount(context.Background(), moneyParty, testMarketAsset)
	marginMoneyParty, err := eng.CreatePartyMarginAccount(context.Background(), moneyParty, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.IncrementBalance(context.Background(), marginMoneyParty, num.UintZero().Mul(price, num.NewUint(5)))
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeMTMLoss,
		},
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeMTMLoss,
		},
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeMTMWin,
		},
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeMTMWin,
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

	// there should be 3 ledger moves -> settle to party 1, settle to party 2, insurance to party 2
	assert.Equal(t, 1, len(raw[1].Entries))
}

func testRemoveDistressedBalance(t *testing.T) {
	party := "test-party"

	insBalance := num.NewUint(1000)
	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, insBalance)
	assert.Nil(t, err)

	// create party accounts (calls buf.Add twice), and add balance (calls it a third time)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	marginID, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// add balance to margin account for party
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.IncrementBalance(context.Background(), marginID, num.NewUint(100))
	assert.Nil(t, err)

	// events:
	data := []events.MarketPosition{
		marketPositionFake{
			party: party,
		},
	}
	eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Id == marginID {
			assert.Zero(t, stringToInt(acc.Balance))
		} else {
			// this doesn't happen yet
			assert.Equal(t, num.UintZero().Add(insBalance, num.NewUint(100)).String(), acc.Balance)
		}
	})
	resp, err := eng.RemoveDistressed(context.Background(), data, testMarketID, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.Entries))

	// check if account was deleted
	_, err = eng.GetAccountByID(marginID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account does not exist:")
}

func testRemoveDistressedNoBalance(t *testing.T) {
	party := "test-party"

	insBalance := num.NewUint(1000)
	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, insBalance)
	assert.Nil(t, err)

	// create party accounts (calls buf.Add twice), and add balance (calls it a third time)
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	marginID, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	// no balance on margin account, so we don't expect there to be any balance updates in the buffer either
	// set up calls expected to buffer: add the update of the balance, of system account (insurance) and one with the margin account set to 0
	data := []events.MarketPosition{
		marketPositionFake{
			party: party,
		},
	}
	resp, err := eng.RemoveDistressed(context.Background(), data, testMarketID, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(resp.Entries))

	// check if account was deleted
	_, err = eng.GetAccountByID(marginID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account does not exist:")
}

// most of this function is copied from the MarkToMarket test - we're using channels, sure
// but the flow should remain the same regardless.
func testMTMSuccess(t *testing.T) {
	party := "test-party"
	moneyParty := "money-party"
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create party accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(8)
	gID, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	mID, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	assert.NotEmpty(t, mID)
	assert.NotEmpty(t, gID)

	// create + add balance
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), moneyParty, testMarketAsset)
	marginMoneyParty, err := eng.CreatePartyMarginAccount(context.Background(), moneyParty, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.UpdateBalance(context.Background(), marginMoneyParty, num.UintZero().Mul(num.NewUint(5), price))
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeMTMLoss,
		},
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeMTMLoss,
		},
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeMTMWin,
		},
		{
			Owner: moneyParty,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeMTMWin,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == party && acc.Type == types.AccountTypeGeneral {
			assert.Equal(t, acc.Balance, int64(833))
		}
		if acc.Owner == moneyParty && acc.Type == types.AccountTypeGeneral {
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
	party := "test-party"
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create party accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	_, err = eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeMTMLoss,
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
	party := "test-party"
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create party accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	_, err = eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: num.UintZero(),
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeMTMLoss,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestNoMarginAccount(t *testing.T) {
	party := "test-party"
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	// create party accounts
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)

	pos := []*types.Transfer{
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeMTMLoss,
		},
	}
	transfers := eng.getTestMTMTransfer(pos)

	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.Error(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Empty(t, evts)
}

func TestNoGeneralAccount(t *testing.T) {
	party := "test-party"
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeMTMLoss,
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

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
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
		party: "test-party",
	}
	transfers = append(transfers, mt)
	evts, raw, err = eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(raw))
	assert.Equal(t, len(evts), 1)
}

func TestFinalSettlementNoTransfers(t *testing.T) {
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	pos := []*types.Transfer{}

	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos, num.UintOne())
	assert.NoError(t, err)
	assert.Equal(t, 0, len(responses))
}

func TestFinalSettlementNoSystemAccounts(t *testing.T) {
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: "testParty",
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferTypeLoss,
		},
	}

	responses, err := eng.FinalSettlement(context.Background(), "invalidMarketID", pos, num.UintOne())
	assert.Error(t, err)
	assert.Equal(t, 0, len(responses))
}

func TestFinalSettlementNotEnoughMargin(t *testing.T) {
	amount := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(amount, num.NewUint(2)))
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), "testParty", testMarketAsset)
	_, err = eng.CreatePartyMarginAccount(context.Background(), "testParty", testMarketID, testMarketAsset)
	require.NoError(t, err)

	pos := []*types.Transfer{
		{
			Owner: "testParty",
			Amount: &types.FinancialAmount{
				Amount: num.UintZero().Mul(amount, num.NewUint(100)),
				Asset:  "BTC",
			},
			Type: types.TransferTypeLoss,
		},
		{
			Owner: "testParty",
			Amount: &types.FinancialAmount{
				Amount: num.UintZero().Mul(amount, num.NewUint(100)),
				Asset:  "BTC",
			},
			Type: types.TransferTypeWin,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	responses, err := eng.FinalSettlement(context.Background(), testMarketID, pos, num.UintOne())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(responses))

	assert.Equal(t, 1, len(responses[0].Entries))
	assert.Equal(t, num.NewUint(500), responses[0].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(500), responses[0].Entries[0].ToAccountBalance)

	assert.Equal(t, 1, len(responses[1].Entries))
	assert.Equal(t, num.NewUint(500), responses[1].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(500), responses[1].Entries[0].ToAccountBalance)
}

func TestGetPartyMarginNoAccounts(t *testing.T) {
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	marketPos := mtmFake{
		party: "test-party",
	}

	margin, err := eng.GetPartyMargin(marketPos, "BTC", testMarketID)
	assert.Nil(t, margin)
	assert.Error(t, err)
}

func TestGetPartyMarginNoMarginAccounts(t *testing.T) {
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), "test-party", testMarketAsset)

	marketPos := mtmFake{
		party: "test-party",
	}

	margin, err := eng.GetPartyMargin(marketPos, "BTC", testMarketID)
	assert.Nil(t, margin)
	assert.Error(t, err)
}

func TestGetPartyMarginEmpty(t *testing.T) {
	price := num.NewUint(1000)

	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	insurancePool, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.Nil(t, err)
	err = eng.UpdateBalance(context.Background(), insurancePool.ID, num.UintZero().Div(price, num.NewUint(2)))
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), "test-party", testMarketAsset)
	_, err = eng.CreatePartyMarginAccount(context.Background(), "test-party", testMarketID, testMarketAsset)
	require.NoError(t, err)

	marketPos := mtmFake{
		party: "test-party",
	}

	margin, err := eng.GetPartyMargin(marketPos, "BTC", testMarketID)
	assert.NotNil(t, margin)
	assert.Equal(t, margin.MarginBalance(), num.UintZero())
	assert.Equal(t, margin.GeneralBalance(), num.UintZero())
	assert.NoError(t, err)
}

func TestMTMLossSocialization(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	lossParty1 := "lossparty1"
	lossParty2 := "lossparty2"
	winParty1 := "winparty1"
	winParty2 := "winparty2"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(18)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), lossParty1, testMarketAsset)
	margin, err := eng.CreatePartyMarginAccount(context.Background(), lossParty1, testMarketID, testMarketAsset)
	eng.IncrementBalance(context.Background(), margin, num.NewUint(500))
	assert.Nil(t, err)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), lossParty2, testMarketAsset)
	margin, err = eng.CreatePartyMarginAccount(context.Background(), lossParty2, testMarketID, testMarketAsset)
	eng.IncrementBalance(context.Background(), margin, num.NewUint(1100))
	assert.Nil(t, err)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), winParty1, testMarketAsset)
	_, err = eng.CreatePartyMarginAccount(context.Background(), winParty1, testMarketID, testMarketAsset)
	// eng.IncrementBalance(context.Background(), margin, 0)
	assert.Nil(t, err)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), winParty2, testMarketAsset)
	_, err = eng.CreatePartyMarginAccount(context.Background(), winParty2, testMarketID, testMarketAsset)
	// eng.IncrementBalance(context.Background(), margin, 700)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: lossParty1,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(700),
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeMTMLoss,
		},
		{
			Owner: lossParty2,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(1400),
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeMTMLoss,
		},
		{
			Owner: winParty1,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(1400),
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeMTMWin,
		},
		{
			Owner: winParty2,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(700),
				Asset:  testMarketAsset,
			},
			Type: types.TransferTypeMTMWin,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == winParty1 && acc.Type == types.AccountTypeMargin {
			assert.Equal(t, 1066, stringToInt(acc.Balance))
		}
		if acc.Owner == winParty2 && acc.Type == types.AccountTypeMargin {
			assert.Equal(t, 534, stringToInt(acc.Balance))
		}
	})
	transfers := eng.getTestMTMTransfer(pos)
	evts, raw, err := eng.MarkToMarket(context.Background(), testMarketID, transfers, "BTC")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(raw))
	assert.NotEmpty(t, evts)

	assert.Equal(t, 1, len(raw[0].Entries))
	assert.Equal(t, num.NewUint(500), raw[0].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(500), raw[0].Entries[0].ToAccountBalance)

	assert.Equal(t, 1, len(raw[1].Entries))
	assert.Equal(t, num.NewUint(1600), raw[1].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(1600), raw[1].Entries[0].ToAccountBalance)

	assert.Equal(t, 1, len(raw[2].Entries))
	assert.Equal(t, num.NewUint(1066), raw[2].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(1066), raw[2].Entries[0].ToAccountBalance)

	assert.Equal(t, 1, len(raw[3].Entries))
	assert.Equal(t, num.NewUint(534), raw[3].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(534), raw[3].Entries[0].ToAccountBalance)
}

func testMarginUpdateOnOrderOK(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	acc, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	eng.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100),
		transfer: &types.Transfer{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100),
			Type:      types.TransferTypeMarginLow,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == party && acc.Type == types.AccountTypeMargin {
			assert.Equal(t, stringToInt(acc.Balance), 100)
		}
	})
	resp, closed, err := eng.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.Nil(t, err)
	assert.Nil(t, closed)
	assert.NotNil(t, resp)

	assert.Equal(t, 1, len(resp.Entries))
	assert.Equal(t, num.NewUint(100), resp.Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(100), resp.Entries[0].ToAccountBalance)
}

func testMarginUpdateOnOrderOKNotShortFallWithBondAccount(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	acc, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	eng.IncrementBalance(context.Background(), acc, num.NewUint(500))
	bondacc, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	eng.IncrementBalance(context.Background(), bondacc, num.NewUint(500))
	_, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100),
		transfer: &types.Transfer{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100),
			Type:      types.TransferTypeMarginLow,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == party && acc.Type == types.AccountTypeMargin {
			assert.Equal(t, stringToInt(acc.Balance), 100)
		}
	})
	resp, closed, err := eng.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.Nil(t, err)
	assert.Nil(t, closed)
	assert.NotNil(t, resp)

	assert.Equal(t, 1, len(resp.Entries))
	assert.Equal(t, num.NewUint(100), resp.Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(100), resp.Entries[0].ToAccountBalance)
}

func testMarginUpdateOnOrderOKUseBondAccount(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	genaccID, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	eng.IncrementBalance(context.Background(), genaccID, num.UintZero())
	bondAccID, _ := eng.CreatePartyBondAccount(context.Background(), party, testMarketID, testMarketAsset)
	eng.IncrementBalance(context.Background(), bondAccID, num.NewUint(500))
	_, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100),
		transfer: &types.Transfer{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100),
			Type:      types.TransferTypeMarginLow,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == party && acc.Type == types.AccountTypeMargin {
			assert.Equal(t, stringToInt(acc.Balance), 100)
		}
	})
	resp, closed, err := eng.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.Nil(t, err)
	assert.NotNil(t, closed)
	assert.NotNil(t, resp)

	assert.Equal(t, closed.MarginShortFall(), num.NewUint(100))

	gacc, err := eng.GetAccountByID(genaccID)
	assert.NoError(t, err)
	assert.Equal(t, num.UintZero(), gacc.Balance)
	bondAcc, err := eng.GetAccountByID(bondAccID)
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(400), bondAcc.Balance)

	assert.Equal(t, 1, len(resp.Entries))
	assert.Equal(t, num.NewUint(100), resp.Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(100), resp.Entries[0].ToAccountBalance)
}

func testMarginUpdateOnOrderOKUseBondAndGeneralAccounts(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	genaccID, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	eng.IncrementBalance(context.Background(), genaccID, num.NewUint(70))
	bondAccID, _ := eng.CreatePartyBondAccount(context.Background(), party, testMarketID, testMarketAsset)
	eng.IncrementBalance(context.Background(), bondAccID, num.NewUint(500))
	marginAccID, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100),
		transfer: &types.Transfer{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100),
			Type:      types.TransferTypeMarginLow,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		// first call is to be updated to 70 with bond accoutns funds
		// then to 100 with general account funds
		if acc.Owner == party && acc.Type == types.AccountTypeMargin {
			assert.True(t, stringToInt(acc.Balance) == 70 || stringToInt(acc.Balance) == 100)
		}
	})

	resp, closed, err := eng.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.Nil(t, err)
	assert.NotNil(t, closed)
	assert.NotNil(t, resp)

	// we toped up only 70 in the bond account
	// but required 100 so we should pick 30 in the general account as well.

	// check shortfall
	assert.Equal(t, closed.MarginShortFall(), num.NewUint(30))

	gacc, err := eng.GetAccountByID(genaccID)
	assert.NoError(t, err)
	assert.Equal(t, num.UintZero(), gacc.Balance)
	bondAcc, err := eng.GetAccountByID(bondAccID)
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(470), bondAcc.Balance)
	marginAcc, err := eng.GetAccountByID(marginAccID)
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(100), marginAcc.Balance)
}

func testMarginUpdateOnOrderOKThenRollback(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	acc, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	eng.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100),
		transfer: &types.Transfer{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100),
			Type:      types.TransferTypeMarginLow,
		},
	}

	eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == party && acc.Type == types.AccountTypeMargin {
			assert.Equal(t, stringToInt(acc.Balance), 100)
		}
		if acc.Owner == party && acc.Type == types.AccountTypeGeneral {
			assert.Equal(t, stringToInt(acc.Balance), 400)
		}
	})
	resp, closed, err := eng.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.Nil(t, err)
	assert.Nil(t, closed)
	assert.NotNil(t, resp)

	// then rollback
	rollback := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: num.NewUint(100),
			Asset:  testMarketAsset,
		},
		MinAmount: num.NewUint(100),
		Type:      types.TransferTypeMarginLow,
	}

	eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Owner == party && acc.Type == types.AccountTypeMargin {
			assert.Equal(t, stringToInt(acc.Balance), 0)
		}
		if acc.Owner == party && acc.Type == types.AccountTypeGeneral {
			assert.Equal(t, stringToInt(acc.Balance), 500)
		}
	})
	resp, err = eng.RollbackMarginUpdateOnOrder(context.Background(), testMarketID, testMarketAsset, rollback)
	assert.Nil(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, 1, len(resp.Entries))
	assert.Equal(t, num.NewUint(500), resp.Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(500), resp.Entries[0].ToAccountBalance)
}

func testMarginUpdateOnOrderFail(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	_, _ = eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	_, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	evt := riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100000),
		transfer: &types.Transfer{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100000),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100000),
			Type:      types.TransferTypeMarginLow,
		},
	}

	resp, closed, err := eng.MarginUpdateOnOrder(context.Background(), testMarketID, evt)
	assert.NotNil(t, err)
	assert.Error(t, err, collateral.ErrMinAmountNotReached.Error())
	assert.NotNil(t, closed)
	assert.Nil(t, resp)
}

func TestMarginUpdates(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(6)
	acc, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	eng.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	list := make([]events.Risk, 1)

	list[0] = riskFake{
		asset:  testMarketAsset,
		amount: num.NewUint(100),
		transfer: &types.Transfer{
			Owner: party,
			Amount: &types.FinancialAmount{
				Amount: num.NewUint(100),
				Asset:  testMarketAsset,
			},
			MinAmount: num.NewUint(100),
			Type:      types.TransferTypeMarginLow,
		},
	}

	resp, margin, _, err := eng.MarginUpdate(context.Background(), testMarketID, list)
	assert.Nil(t, err)
	assert.Equal(t, len(margin), 0)
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Entries[0].Amount, num.NewUint(100))

	assert.Equal(t, 1, len(resp[0].Entries))
	assert.Equal(t, num.NewUint(100), resp[0].Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(100), resp[0].Entries[0].ToAccountBalance)
}

func TestClearMarket(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(12)

	eng.IncrementBalance(context.Background(), eng.marketInsuranceID, num.NewUint(1000))

	_, err := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	assert.Nil(t, err)
	acc, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	eng.IncrementBalance(context.Background(), acc, num.NewUint(500))
	assert.Nil(t, err)

	// increment the balance on the lpFee account so we can check it gets cleared
	liqAcc, _ := eng.GetMarketLiquidityFeeAccount(testMarketID, testMarketAsset)
	eng.IncrementBalance(context.Background(), liqAcc.ID, num.NewUint(250))

	parties := []string{party}

	responses, err := eng.ClearMarket(context.Background(), testMarketID, testMarketAsset, parties, false)

	assert.Nil(t, err)
	assert.Equal(t, 3, len(responses))

	// this will be from the margin account to the general account
	assert.Equal(t, 1, len(responses[0].Entries))
	entry := responses[0].Entries[0]
	assert.Equal(t, types.AccountTypeMargin, entry.FromAccount.Type)
	assert.Equal(t, types.AccountTypeGeneral, entry.ToAccount.Type)
	assert.Equal(t, num.NewUint(0), entry.FromAccountBalance)
	assert.Equal(t, num.NewUint(500), entry.ToAccountBalance)
	assert.Equal(t, num.NewUint(500), entry.Amount)

	// This will be liquidity fees being cleared into the insurance account
	assert.Equal(t, 1, len(responses[1].Entries))
	entry = responses[1].Entries[0]
	assert.Equal(t, types.AccountTypeFeesLiquidity, entry.FromAccount.Type)
	assert.Equal(t, types.AccountTypeInsurance, entry.ToAccount.Type)
	assert.Equal(t, num.NewUint(0), entry.FromAccountBalance)
	assert.Equal(t, num.NewUint(1250), entry.ToAccountBalance)
	assert.Equal(t, num.NewUint(250), entry.Amount)

	// This will be the insurance account going into the global insurance pool
	entry = responses[2].Entries[0]
	assert.Equal(t, types.AccountTypeInsurance, entry.FromAccount.Type)
	assert.Equal(t, types.AccountTypeGlobalInsurance, entry.ToAccount.Type)
	assert.Equal(t, num.NewUint(0), entry.FromAccountBalance)
	assert.Equal(t, num.NewUint(1250), entry.ToAccountBalance)
	assert.Equal(t, num.NewUint(1250), entry.Amount)
}

func TestClearMarketNoMargin(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	acc, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	eng.IncrementBalance(context.Background(), acc, num.NewUint(500))

	parties := []string{party}

	responses, err := eng.ClearMarket(context.Background(), testMarketID, testMarketAsset, parties, false)

	// we expect no ledger movements as all accounts to clear were empty
	assert.NoError(t, err)
	assert.Equal(t, len(responses), 0)
}

func TestRewardDepositOK(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	ctx := context.Background()

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Attempt to deposit collateral that should go into the global asset reward account
	_, err := eng.Deposit(ctx, rewardsID, testMarketAsset, num.NewUint(100))
	assert.NoError(t, err)

	rewardAcct, err := eng.GetGlobalRewardAccount(testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(100), rewardAcct.Balance)

	// Add 400 more to the reward account
	_, err = eng.Deposit(ctx, rewardsID, testMarketAsset, num.NewUint(400))
	assert.NoError(t, err)

	rewardAcct, err = eng.GetGlobalRewardAccount(testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, num.NewUint(500), rewardAcct.Balance)
}

func TestNonRewardDepositOK(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	ctx := context.Background()

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Attempt to deposit collateral that should go into the global asset reward account
	_, err := eng.Deposit(ctx, "OtherParty", testMarketAsset, num.NewUint(100))
	assert.NoError(t, err)
}

func TestRewardDepositBadAssetOK(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	ctx := context.Background()
	testAsset2 := "VEGA"

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Now try a different asset
	_, err := eng.Deposit(ctx, rewardsID, testAsset2, num.NewUint(333))
	assert.Error(t, err)
}

func TestWithdrawalOK(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	acc, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	eng.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		ae, ok := evt.(accEvt)
		assert.True(t, ok)
		acc := ae.Account()
		if acc.Type == types.AccountTypeGeneral {
			assert.Equal(t, 400, stringToInt(acc.Balance))
		} else {
			t.FailNow()
		}
	})

	lm, err := eng.Withdraw(context.Background(), party, testMarketAsset, num.NewUint(100))
	assert.Nil(t, err)

	assert.Equal(t, 1, len(lm.Entries))
	assert.Equal(t, num.NewUint(100), lm.Entries[0].ToAccountBalance)
	assert.Equal(t, num.NewUint(100), lm.Entries[0].ToAccountBalance)
}

func TestWithdrawalExact(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(5)
	acc, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	eng.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_, err = eng.Withdraw(context.Background(), party, testMarketAsset, num.NewUint(500))
	assert.Nil(t, err)

	accAfter, err := eng.GetPartyGeneralAccount(party, testMarketAsset)
	assert.NoError(t, err)
	assert.Equal(t, accAfter.Balance, num.UintZero())
}

func TestWithdrawalNotEnough(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	acc, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	eng.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_, err = eng.Withdraw(context.Background(), party, testMarketAsset, num.NewUint(600))
	assert.EqualError(t, err, collateral.ErrNotEnoughFundsToWithdraw.Error())
}

func TestWithdrawalInvalidAccount(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	// create parties
	eng.broker.EXPECT().Send(gomock.Any()).Times(4)
	acc, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)
	eng.IncrementBalance(context.Background(), acc, num.NewUint(500))
	_, err := eng.CreatePartyMarginAccount(context.Background(), party, testMarketID, testMarketAsset)
	assert.Nil(t, err)

	_, err = eng.Withdraw(context.Background(), "invalid", testMarketAsset, num.NewUint(600))
	assert.Error(t, err)
}

func TestChangeBalance(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	party := "okparty"

	eng.broker.EXPECT().Send(gomock.Any()).Times(2)
	acc, _ := eng.CreatePartyGeneralAccount(context.Background(), party, testMarketAsset)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	eng.IncrementBalance(context.Background(), acc, num.NewUint(500))
	account, err := eng.GetAccountByID(acc)
	assert.NoError(t, err)
	assert.Equal(t, account.Balance, num.NewUint(500))

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	eng.IncrementBalance(context.Background(), acc, num.NewUint(250))
	account, err = eng.GetAccountByID(acc)
	require.NoError(t, err)
	assert.Equal(t, account.Balance, num.NewUint(750))

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	eng.UpdateBalance(context.Background(), acc, num.NewUint(666))
	account, err = eng.GetAccountByID(acc)
	require.NoError(t, err)
	assert.Equal(t, account.Balance, num.NewUint(666))

	err = eng.IncrementBalance(context.Background(), "invalid", num.NewUint(200))
	assert.Error(t, err, collateral.ErrAccountDoesNotExist)

	err = eng.UpdateBalance(context.Background(), "invalid", num.NewUint(300))
	assert.Error(t, err, collateral.ErrAccountDoesNotExist)
}

func TestReloadConfig(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	// Check that the log level is currently `debug`
	assert.Equal(t, eng.Level.Level, logging.DebugLevel)

	// Create a new config and make some changes to it
	newConfig := collateral.NewDefaultConfig()
	newConfig.Level = encoding.LogLevel{
		Level: logging.InfoLevel,
	}
	eng.ReloadConf(newConfig)

	// Verify that the log level has been changed
	assert.Equal(t, eng.Level.Level, logging.InfoLevel)
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

func enableGovernanceAsset(t *testing.T, eng *collateral.Engine) {
	t.Helper()
	// add the token asset
	tokAsset := types.Asset{
		ID: "VOTE",
		Details: &types.AssetDetails{
			Name:     "VOTE",
			Symbol:   "VOTE",
			Decimals: 5,
			Quantum:  num.DecimalZero(),
			Source: &types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.UintZero(),
				},
			},
		},
	}
	err := eng.EnableAsset(context.Background(), tokAsset)
	assert.NoError(t, err)
}

func getTestEngine(t *testing.T) *testEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	timeSvc := mocks.NewMockTimeService(ctrl)
	timeSvc.EXPECT().GetTimeNow().AnyTimes()

	broker := bmocks.NewMockBroker(ctrl)
	conf := collateral.NewDefaultConfig()
	conf.Level = encoding.LogLevel{Level: logging.DebugLevel}
	broker.EXPECT().Send(gomock.Any()).Times(22)
	// system accounts created

	eng := collateral.New(logging.NewTestLogger(), conf, timeSvc, broker)

	enableGovernanceAsset(t, eng)

	// enable the assert for the tests
	asset := types.Asset{
		ID: testMarketAsset,
		Details: &types.AssetDetails{
			Symbol:   testMarketAsset,
			Name:     testMarketAsset,
			Decimals: 0,
			Quantum:  num.DecimalZero(),
			Source: &types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.UintZero(),
				},
			},
		},
	}
	err := eng.EnableAsset(context.Background(), asset)
	assert.NoError(t, err)
	// ETH is added hardcoded in some places
	asset = types.Asset{
		ID: "ETH",
		Details: &types.AssetDetails{
			Symbol:   "ETH",
			Name:     "ETH",
			Decimals: 18,
			Quantum:  num.DecimalZero(),
			Source: &types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.UintZero(),
				},
			},
		},
	}
	err = eng.EnableAsset(context.Background(), asset)
	assert.NoError(t, err)

	// create market and parties used for tests
	insID, setID, err := eng.CreateMarketAccounts(context.Background(), testMarketID, testMarketAsset)
	assert.Nil(t, err)

	return &testEngine{
		Engine:             eng,
		ctrl:               ctrl,
		broker:             broker,
		timeSvc:            timeSvc,
		marketInsuranceID:  insID,
		marketSettlementID: setID,
		// systemAccs: accounts,
	}
}

func TestCheckLeftOverBalance(t *testing.T) {
	e := getTestEngine(t)
	defer e.Finish()

	e.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	ctx := context.Background()
	marketID := crypto.RandomHash()
	asset := "ETH"
	settleAccountID, _, err := e.CreateMarketAccounts(ctx, marketID, asset)
	require.NoError(t, err)

	// settle account is empty, all good, no error, no leftover ledger entry
	settle := &types.Account{
		ID:      settleAccountID,
		Balance: num.UintZero(),
	}
	leftoverTransfer, err := e.CheckLeftOverBalance(ctx, settle, []*types.Transfer{}, asset, num.UintOne())
	require.NoError(t, err)
	require.Nil(t, leftoverTransfer)

	// settle has balance greater than 1, panic
	settle.Balance = num.NewUint(100)
	require.Panics(t, func() { e.CheckLeftOverBalance(ctx, settle, []*types.Transfer{}, asset, num.UintOne()) })

	// settle has balance greater than 1, market factor of 10, still panic
	settle.Balance = num.NewUint(100)
	require.Panics(t, func() { e.CheckLeftOverBalance(ctx, settle, []*types.Transfer{}, asset, num.NewUint(10)) })

	// settle has balance greater than 1, for a market with price factor 1000 is fine
	settle.Balance = num.NewUint(100)
	leftoverTransfer, err = e.CheckLeftOverBalance(ctx, settle, []*types.Transfer{}, asset, num.NewUint(1000))
	require.NoError(t, err)
	require.NotNil(t, leftoverTransfer)

	// settle has balance of exactly 1, transfer balance to the reward account
	settle.Balance = num.NewUint(1)
	leftoverTransfer, err = e.CheckLeftOverBalance(ctx, settle, []*types.Transfer{}, asset, num.UintOne())
	require.NoError(t, err)
	require.NotNil(t, leftoverTransfer)
}

func (e *testEngine) Finish() {
	e.systemAccs = nil
	e.ctrl.Finish()
}

type marketPositionFake struct {
	party                         string
	size, buy, sell               int64
	price                         *num.Uint
	buySumProduct, sellSumProduct *num.Uint
}

func (m marketPositionFake) Party() string             { return m.party }
func (m marketPositionFake) Size() int64               { return m.size }
func (m marketPositionFake) Buy() int64                { return m.buy }
func (m marketPositionFake) Sell() int64               { return m.sell }
func (m marketPositionFake) Price() *num.Uint          { return m.price }
func (m marketPositionFake) BuySumProduct() *num.Uint  { return m.buySumProduct }
func (m marketPositionFake) SellSumProduct() *num.Uint { return m.sellSumProduct }
func (m marketPositionFake) ClearPotentials()          {}

func (m marketPositionFake) VWBuy() *num.Uint {
	if m.buy == 0 {
		return num.UintZero()
	}
	return num.UintZero().Div(num.NewUint(uint64(m.buy)), m.buySumProduct)
}

func (m marketPositionFake) VWSell() *num.Uint {
	if m.sell == 0 {
		return num.UintZero()
	}
	return num.UintZero().Div(num.NewUint(uint64(m.sell)), m.sellSumProduct)
}

type mtmFake struct {
	t     *types.Transfer
	party string
}

func (m mtmFake) Party() string             { return m.party }
func (m mtmFake) Size() int64               { return 0 }
func (m mtmFake) Price() *num.Uint          { return num.UintZero() }
func (m mtmFake) BuySumProduct() *num.Uint  { return num.UintZero() }
func (m mtmFake) SellSumProduct() *num.Uint { return num.UintZero() }
func (m mtmFake) VWBuy() *num.Uint          { return num.UintZero() }
func (m mtmFake) VWSell() *num.Uint         { return num.UintZero() }
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
	party                         string
	size, buy, sell               int64
	price                         *num.Uint
	buySumProduct, sellSumProduct *num.Uint
	vwBuy, vwSell                 *num.Uint
	margins                       *types.MarginLevels
	amount                        *num.Uint
	transfer                      *types.Transfer
	asset                         string
	marginShortFall               *num.Uint
}

func (m riskFake) Party() string                     { return m.party }
func (m riskFake) Size() int64                       { return m.size }
func (m riskFake) Buy() int64                        { return m.buy }
func (m riskFake) Sell() int64                       { return m.sell }
func (m riskFake) Price() *num.Uint                  { return m.price }
func (m riskFake) BuySumProduct() *num.Uint          { return m.buySumProduct }
func (m riskFake) SellSumProduct() *num.Uint         { return m.sellSumProduct }
func (m riskFake) VWBuy() *num.Uint                  { return m.vwBuy }
func (m riskFake) VWSell() *num.Uint                 { return m.vwSell }
func (m riskFake) ClearPotentials()                  {}
func (m riskFake) Transfer() *types.Transfer         { return m.transfer }
func (m riskFake) Amount() *num.Uint                 { return m.amount }
func (m riskFake) MarginLevels() *types.MarginLevels { return m.margins }
func (m riskFake) Asset() string                     { return m.asset }
func (m riskFake) MarketID() string                  { return "" }
func (m riskFake) MarginBalance() *num.Uint          { return num.UintZero() }
func (m riskFake) GeneralBalance() *num.Uint         { return num.UintZero() }
func (m riskFake) BondBalance() *num.Uint            { return num.UintZero() }
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
	eng := getTestEngine(t)
	defer eng.Finish()

	// Create the accounts
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	id1, err := eng.CreatePartyGeneralAccount(context.Background(), "t1", testMarketAsset)
	require.NoError(t, err)

	id2, err := eng.CreatePartyGeneralAccount(context.Background(), "t2", testMarketAsset)
	require.NoError(t, err)

	_, err = eng.CreatePartyMarginAccount(context.Background(), "t1", testMarketID, testMarketAsset)
	require.NoError(t, err)

	// Add balances
	require.NoError(t,
		eng.UpdateBalance(context.Background(), id1, num.NewUint(100)),
	)

	require.NoError(t,
		eng.UpdateBalance(context.Background(), id2, num.NewUint(500)),
	)

	hash := eng.Hash()
	require.Equal(t,
		"589c48274f3ab644f725d9abc4de9cb07b6ea9069dd3bd8f41f35dc55d062550",
		hex.EncodeToString(hash),
		"It should match against the known hash",
	)
	// compute the hash 100 times for determinism verification
	for i := 0; i < 100; i++ {
		got := eng.Hash()
		require.Equal(t, hash, got)
	}
}

func stringToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func TestHoldingAccount(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	ctx := context.Background()

	// create the general account for the source general account
	id, err := eng.CreatePartyGeneralAccount(ctx, "zohar", "BTC")
	require.NoError(t, err)

	// topup the source general account
	require.NoError(t, eng.IncrementBalance(ctx, id, num.NewUint(1000)))

	// we're have 1000 in the general account and 0 in the holding
	// transferring 800 from the general account to the holding account in two transfers of 400
	// expect to have the holding account balance = 800 and the general account balance = 200

	// holding account does not exist yet - it will be created
	le, err := eng.TransferToHoldingAccount(ctx, &types.Transfer{
		Owner: "zohar",
		Amount: &types.FinancialAmount{
			Asset:  "BTC",
			Amount: num.NewUint(400),
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.AccountTypeHolding, le.Balances[0].Account.Type)
	require.Equal(t, num.NewUint(400), le.Balances[0].Balance)

	// holding account does not exist yet - it will be created
	le, err = eng.TransferToHoldingAccount(ctx, &types.Transfer{
		Owner: "zohar",
		Amount: &types.FinancialAmount{
			Asset:  "BTC",
			Amount: num.NewUint(400),
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.AccountTypeHolding, le.Balances[0].Account.Type)
	require.Equal(t, num.NewUint(400), le.Balances[0].Balance)

	// check general account balance is 200
	z, err := eng.GetPartyGeneralAccount("zohar", "BTC")
	require.NoError(t, err)
	require.Equal(t, num.NewUint(200), z.Balance)

	// request to release 200 from the holding account
	le, err = eng.ReleaseFromHoldingAccount(ctx, &types.Transfer{
		Owner: "zohar",
		Amount: &types.FinancialAmount{
			Asset:  "BTC",
			Amount: num.NewUint(200),
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.AccountTypeGeneral, le.Balances[0].Account.Type)
	require.Equal(t, num.NewUint(200), le.Balances[0].Balance)

	// now request to release 600 more
	le, err = eng.ReleaseFromHoldingAccount(ctx, &types.Transfer{
		Owner: "zohar",
		Amount: &types.FinancialAmount{
			Asset:  "BTC",
			Amount: num.NewUint(600),
		},
	})

	require.NoError(t, err)
	require.Equal(t, num.UintZero(), le.Entries[0].FromAccountBalance)
	require.Equal(t, num.NewUint(1000), le.Entries[0].ToAccountBalance)
}

func TestClearSpotMarket(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// create a spot market and top up the fees account before we try to close the market
	err := eng.CreateSpotMarketAccounts(context.Background(), testMarketID, "BTC")
	require.NoError(t, err)

	acc, err := eng.GetMarketLiquidityFeeAccount(testMarketID, "BTC")
	require.NoError(t, err)

	eng.IncrementBalance(context.Background(), acc.ID, num.NewUint(1000))

	_, err = eng.GetMarketMakerFeeAccount(testMarketID, "BTC")
	require.NoError(t, err)

	_, err = eng.ClearSpotMarket(context.Background(), testMarketID, "BTC")
	require.NoError(t, err)

	treasury, err := eng.GetNetworkTreasuryAccount("BTC")
	require.NoError(t, err)
	require.Equal(t, num.NewUint(1000), treasury.Balance)

	// the liquidity and makes fees should be removed at this point
	_, err = eng.GetMarketLiquidityFeeAccount(testMarketID, "BTC")
	require.Error(t, err)

	_, err = eng.GetMarketMakerFeeAccount(testMarketID, "BTC")
	require.Error(t, err)
}

func TestCreateSpotMarketAccounts(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	err := eng.CreateSpotMarketAccounts(context.Background(), testMarketID, "BTC")
	require.NoError(t, err)

	// check that accounts were created for liquidity and maker fees
	_, err = eng.GetMarketLiquidityFeeAccount(testMarketID, "BTC")
	require.NoError(t, err)

	_, err = eng.GetMarketMakerFeeAccount(testMarketID, "BTC")
	require.NoError(t, err)
}

func TestPartyHasSufficientBalance(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// first check when general account of the source does not exist
	err := eng.PartyHasSufficientBalance("BTC", "zohar", num.NewUint(1000))
	require.Error(t, err)

	ctx := context.Background()
	// create the general account for the source general account
	id, err := eng.CreatePartyGeneralAccount(ctx, "zohar", "BTC")
	require.NoError(t, err)

	// topup the source general account
	require.NoError(t, eng.IncrementBalance(ctx, id, num.NewUint(1000)))

	err = eng.PartyHasSufficientBalance("BTC", "zohar", num.NewUint(1001))
	require.Error(t, err)
	err = eng.PartyHasSufficientBalance("BTC", "zohar", num.NewUint(1000))
	require.NoError(t, err)
	err = eng.PartyHasSufficientBalance("BTC", "zohar", num.NewUint(900))
	require.NoError(t, err)
}

func TestCreatePartyHoldingAccount(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	ctx := context.Background()

	_, err := eng.CreatePartyHoldingAccount(ctx, "BTC2", "zohar")
	// asset does not exist
	require.Error(t, err)

	id, err := eng.CreatePartyHoldingAccount(ctx, "zohar", "BTC")
	require.NoError(t, err)

	eng.IncrementBalance(ctx, id, num.NewUint(1000))

	// check holding account balance
	acc, err := eng.GetAccountByID(id)
	require.NoError(t, err)
	require.Equal(t, types.AccountTypeHolding, acc.Type)
	require.Equal(t, num.NewUint(1000), acc.Balance)
}

func TestTransferSpot(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	ctx := context.Background()
	// first check when general account of the source does not exist
	_, err := eng.TransferSpot(ctx, "zohar", "jeremy", "BTC", num.NewUint(900))
	require.Error(t, err)

	// create the general account for the source general account
	id, err := eng.CreatePartyGeneralAccount(ctx, "zohar", "BTC")
	require.NoError(t, err)

	// topup the source general account
	require.NoError(t, eng.IncrementBalance(ctx, id, num.NewUint(1000)))

	// transfer successfully
	_, err = eng.TransferSpot(ctx, "zohar", "jeremy", "BTC", num.NewUint(900))
	require.NoError(t, err)

	// check balances
	z, err := eng.GetPartyGeneralAccount("zohar", "BTC")
	require.NoError(t, err)

	j, err := eng.GetPartyGeneralAccount("jeremy", "BTC")
	require.NoError(t, err)

	require.Equal(t, num.NewUint(100), z.Balance)
	require.Equal(t, num.NewUint(900), j.Balance)

	// try to transfer more than in the account should transfer all
	_, err = eng.TransferSpot(ctx, "jeremy", "zohar", "BTC", num.NewUint(1000))
	require.NoError(t, err)

	// check balances
	z, err = eng.GetPartyGeneralAccount("zohar", "BTC")
	require.NoError(t, err)

	j, err = eng.GetPartyGeneralAccount("jeremy", "BTC")
	require.NoError(t, err)

	require.Equal(t, num.NewUint(1000), z.Balance)
	require.Equal(t, num.UintZero(), j.Balance)
}
