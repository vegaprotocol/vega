package collateral_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/internal/collateral"
	"code.vegaprotocol.io/vega/internal/collateral/mocks"
	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	testMarketID    = "7CPSHJB35AIQBTNMIE6NLFPZGHOYRQ3D"
	testMarketAsset = "BTC"
)

var (
	generalSystem = types.Account{
		Id:      "system-gen",
		Owner:   storage.SystemOwner,
		Balance: 0,
		Asset:   "BTC",
		Type:    types.AccountType_GENERAL,
	}

	settlementSystem = types.Account{
		Id:      "system-set",
		Owner:   storage.SystemOwner,
		Balance: 0,
		Asset:   "BTC",
		Type:    types.AccountType_SETTLEMENT,
	}

	insuranceSystem = types.Account{
		Id:      "system-ins",
		Owner:   storage.SystemOwner,
		Balance: 0,
		Asset:   "BTC",
		Type:    types.AccountType_INSURANCE,
	}
)

type testEngine struct {
	*collateral.Engine
	ctrl               *gomock.Controller
	buf                *mocks.MockAccountBuffer
	accounts           *mocks.MockAccounts
	systemAccs         []*types.Account
	marketInsuranceID  string
	marketSettlementID string
}

func TestCollateralTransfer(t *testing.T) {
	t.Run("test creating new - should create market accounts", testNew)
	t.Run("test collecting buys - both insurance and sufficient in trader accounts", testTransferLoss)
	t.Run("test collecting buys - trader account not empty, but insufficient", testTransferComplexLoss)
	// t.Run("test collecting buys - trader missing some accounts", testTransferLossMissingTraderAccounts)
	// t.Run("test collecting sells - cases where settle account is full + where insurance pool is tapped", testDistributeWin)
	// t.Run("test collecting both buys and sells - Successfully collect buy and sell in a single call", testProcessBoth)
	// t.Run("test distribution insufficient funds - Transfer losses (partial), distribute wins pro-rate", testProcessBothProRated)
}

/*
func TestCollateralMarkToMarket(t *testing.T) {
	t.Run("Mark to Market distribution, insufficient funcs - complex scenario", testProcessBothProRatedMTM)
}

*/

func TestAddTraderToMarket(t *testing.T) {
	t.Run("Successful calls adding new traders (one duplicate, one actual new)", testAddTrader)
}

func testNew(t *testing.T) {
	eng := getTestEngine(t, "test-market", 0)
	eng.Finish()
}

func testAddTrader(t *testing.T) {
	eng := getTestEngine(t, testMarketID, 0)
	defer eng.Finish()
	trader := "funkytrader"

	// create trder
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	margin, general := eng.Engine.CreateTraderAccount(trader, testMarketID, testMarketAsset)

	// add funds
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	eng.Engine.UpdateBalance(general, eng.Config.TraderGeneralAccountBalance)

	// add to the market
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	err := eng.Engine.AddTraderToMarket(testMarketID, trader, testMarketAsset)

	expectedMarginBalance := int64(eng.Config.TraderGeneralAccountBalance / 100 * eng.Config.TraderMarginPercent)
	expectedGeneralBalance := eng.Config.TraderGeneralAccountBalance - expectedMarginBalance

	// check the amount on each account now
	acc, err := eng.Engine.GetAccountByID(margin)
	assert.Nil(t, err)
	assert.Equal(t, acc.Balance, expectedMarginBalance)

	acc, err = eng.Engine.GetAccountByID(general)
	assert.Nil(t, err)
	assert.Equal(t, acc.Balance, expectedGeneralBalance)

}

func testTransferLoss(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price*5)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	_, _ = eng.Engine.CreateTraderAccount(trader, testMarketID, testMarketAsset)

	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	marginMoneyTrader, _ := eng.Engine.CreateTraderAccount(moneyTrader, testMarketID, testMarketAsset)
	err := eng.Engine.IncrementBalance(marginMoneyTrader, 100000)
	assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
	}

	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	responses, err := eng.Transfer(testMarketID, pos)
	assert.Equal(t, 1, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance of settlement account should be 3 times price
	assert.Equal(t, 3*price, resp.Balances[0].Balance)
	// there should be 2 ledger moves
	assert.Equal(t, 2, len(resp.Transfers))
}

func testTransferComplexLoss(t *testing.T) {
	trader := "test-trader"
	half := int64(500)
	price := half * 2

	eng := getTestEngine(t, testMarketID, price*5)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	marginTrader, _ := eng.Engine.CreateTraderAccount(trader, testMarketID, testMarketAsset)
	err := eng.Engine.IncrementBalance(marginTrader, half)
	assert.Nil(t, err)

	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Asset:  "BTC",
				Amount: -price,
			},
			Type: types.TransferType_LOSS,
		},
	}
	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	responses, err := eng.Transfer(testMarketID, pos)
	assert.Equal(t, 1, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance should equal price (only 1 call after all)
	assert.Equal(t, price, resp.Balances[0].Balance)
	// there should be 2 ledger moves, one from trader account, one from insurance acc
	assert.Equal(t, 2, len(resp.Transfers))
}

/*

func testTransferLossMissingTraderAccounts(t *testing.T) {
	market := "BTCtest-market"
	trader := "test-trader"
	price := int64(1000)
	eng := getTestEngine(t, market, nil)
	defer eng.Finish()

	allAccs := getTraderAccounts(trader, market)
	traderAccs := make([]*types.Account, 0, len(allAccs)-1)
	for _, acc := range allAccs {
		// all but margin account
		if acc.Type != types.AccountType_MARGIN {
			traderAccs = append(traderAccs, acc)
		}
	}
	systemAccs := eng.systemAccs
	// we're going to auto-create the accounts
	eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(1).DoAndReturn(func(owner, market string) ([]*types.Account, error) {
		assert.Equal(t, trader, owner)
		return traderAccs, nil
	})
	// set up the get-accounts calls
	for _, acc := range systemAccs {
		if acc.Type == types.AccountType_INSURANCE || acc.Type == types.AccountType_SETTLEMENT {
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
		}
	}
	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Asset:  "BTC",
				Amount: -price,
			},
			Type: types.TransferType_LOSS,
		},
	}
	resp, err := eng.Transfer(pos)
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Equal(t, collateral.ErrTraderAccountsMissing, err)
}

func testDistributeWin(t *testing.T) {
	market := "BTCtest-market"
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	eng := getTestEngine(t, market, nil)
	defer eng.Finish()

	systemAccs := eng.systemAccs
	traderAccs := getTraderAccounts(trader, market)
	moneyAccs := getTraderAccounts(moneyTrader, market)
	// we're going to auto-create the accounts
	eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(2).DoAndReturn(func(owner, market string) ([]*types.Account, error) {
		isTrader := (owner == trader || owner == moneyTrader)
		assert.True(t, isTrader)
		if owner == trader {
			return traderAccs, nil
		}
		return moneyAccs, nil
	})
	// system accounts
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
			// insurance will be used to settle one sale (size 1, of value price, taken from insurance account)
			sacc.Balance = price
			eng.accounts.EXPECT().GetAccountByID(sacc.Id).Times(1).Return(sacc, nil)
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, -price).Times(1).Return(nil)
		case types.AccountType_SETTLEMENT:
			// assign to var so we don't need to repeat this loop for sells
			sacc.Balance = 2 * price
			eng.accounts.EXPECT().GetAccountByID(sacc.Id).Times(1).Return(sacc, nil)
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, -price).Times(1).Return(nil)
			eng.accounts.EXPECT().UpdateBalance(sacc.Id, gomock.Any()).Times(1).Return(nil)
		}
	}
	// now settlement for buys on trader with money:
	for _, acc := range moneyAccs {
		switch acc.Type {
		case types.AccountType_MARGIN:
			acc.Balance += 5 * price
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil)
		case types.AccountType_GENERAL:
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil)
			eng.accounts.EXPECT().IncrementBalance(acc.Id, 2*price).Times(1).Return(nil)
		}
	}
	for _, acc := range traderAccs {
		switch acc.Type {
		case types.AccountType_GENERAL:
			eng.accounts.EXPECT().IncrementBalance(acc.Id, price).Times(1).Return(nil)
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
			fallthrough
		case types.AccountType_MARGIN:
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil).Do(func(_ string, bal int64) {
				assert.NotZero(t, bal)
			})
		}
	}
	// now the positions
	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_WIN,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_WIN,
		},
	}
	responses, err := eng.Transfer(pos)
	assert.Equal(t, 1, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance of settlement account should be 3 times price
	for _, bal := range resp.Balances {
		if bal.Account.Type == types.AccountType_SETTLEMENT {
			assert.Zero(t, bal.Account.Balance)
		}
	}
	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 3, len(resp.Transfers))
}

func testProcessBoth(t *testing.T) {
	market := "BTCtest-market"
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	eng := getTestEngine(t, market, nil)
	defer eng.Finish()

	systemAccs := eng.systemAccs
	traderAccs := getTraderAccounts(trader, market)
	moneyAccs := getTraderAccounts(moneyTrader, market)
	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_WIN,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_WIN,
		},
	}
	// The, each time we encounter a trader (ie each position aggregate), we'll attempt to create the account
	// create the trader accounts, they'll be returned anyway
	eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(len(pos) / 2).DoAndReturn(func(owner, market string) ([]*types.Account, error) {
		isTrader := (owner == trader || owner == moneyTrader)
		assert.True(t, isTrader)
		if owner == trader {
			return traderAccs, nil
		}
		return moneyAccs, nil
	})
	// system accounts
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
			// insurance will be used to settle one sale (size 1, of value price, taken from insurance account)
			sacc.Balance = price * 3
			eng.accounts.EXPECT().GetAccountByID(sacc.Id).Times(1).Return(sacc, nil)
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, -price).Times(1).Return(nil)
		case types.AccountType_SETTLEMENT:
			// assign to var so we don't need to repeat this loop for sells
			eng.accounts.EXPECT().GetAccountByID(sacc.Id).Times(1).Return(sacc, nil)
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, 3*price).Times(1).Return(nil)
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, gomock.Any()).Times(2).Return(nil).Do(func(_ string, inc int64) {
				assert.NotZero(t, inc)
			})
		}
	}
	// now settlement for buys on trader with money:
	for _, acc := range moneyAccs {
		switch acc.Type {
		case types.AccountType_MARGIN:
			acc.Balance += 5 * price
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil)
			eng.accounts.EXPECT().IncrementBalance(acc.Id, -2*price).Times(1).Return(nil)
		case types.AccountType_GENERAL:
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil)
			eng.accounts.EXPECT().IncrementBalance(acc.Id, 2*price).Times(1).Return(nil)
		}
	}
	for _, acc := range traderAccs {
		switch acc.Type {
		case types.AccountType_GENERAL:
			eng.accounts.EXPECT().IncrementBalance(acc.Id, price).Times(1).Return(nil)
			fallthrough
		case types.AccountType_MARGIN:
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil).Do(func(_ string, bal int64) {
				assert.NotZero(t, bal)
			})
		}
	}
	// next up, updating the balance of the traders' general accounts
	responses, err := eng.Transfer(pos)
	assert.Equal(t, 2, len(responses))
	assert.NoError(t, err)
	resp := responses[0]
	// total balance of settlement account should be 3 times price
	for _, bal := range resp.Balances {
		if bal.Account.Type == types.AccountType_SETTLEMENT {
			assert.Zero(t, bal.Account.Balance)
		}
	}
	resp = responses[1]
	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 2, len(resp.Transfers))
}

func testProcessBothProRated(t *testing.T) {
	market := "BTCtest-market"
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	eng := getTestEngine(t, market, nil)
	defer eng.Finish()

	systemAccs := eng.systemAccs
	traderAccs := getTraderAccounts(trader, market)
	moneyAccs := getTraderAccounts(moneyTrader, market)
	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_LOSS,
		},
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_WIN,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_WIN,
		},
	}
	// The, each time we encounter a trader (ie each position aggregate), we'll attempt to create the account
	eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(len(pos) / 2).DoAndReturn(func(owner, market string) ([]*types.Account, error) {
		isTrader := (owner == trader || owner == moneyTrader)
		assert.True(t, isTrader)
		if owner == trader {
			return traderAccs, nil
		}
		return moneyAccs, nil
	})
	// now the positions, calls we expect to be made when processing buys
	// system accounts
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
			// insurance will be used to settle one sale (size 1, of value price, taken from insurance account)
			sacc.Balance = price / 2
			eng.accounts.EXPECT().GetAccountByID(sacc.Id).Times(1).Return(sacc, nil)
			eng.accounts.EXPECT().UpdateBalance(sacc.Id, gomock.Any()).Times(1).Return(nil)
		case types.AccountType_SETTLEMENT:
			// assign to var so we don't need to repeat this loop for sells
			exp := 2 * price
			exp += price / 2
			eng.accounts.EXPECT().GetAccountByID(sacc.Id).Times(1).Return(sacc, nil)
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, exp).Times(1).Return(nil)
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, gomock.Any()).Times(2).Return(nil).Do(func(_ string, inc int64) {
				assert.NotZero(t, inc)
			})
		}
	}
	// now settlement for buys on trader with money:
	for _, acc := range moneyAccs {
		switch acc.Type {
		case types.AccountType_MARGIN:
			acc.Balance += 5 * price
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil)
			eng.accounts.EXPECT().IncrementBalance(acc.Id, -2*price).Times(1).Return(nil)
		case types.AccountType_GENERAL:
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil)
			eng.accounts.EXPECT().IncrementBalance(acc.Id, int64(1666)).Times(1).Return(nil)
		}
	}
	for _, acc := range traderAccs {
		switch acc.Type {
		case types.AccountType_GENERAL:
			eng.accounts.EXPECT().IncrementBalance(acc.Id, int64(833)).Times(1).Return(nil)
			fallthrough
		case types.AccountType_MARGIN:
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil).Do(func(_ string, bal int64) {
				assert.NotZero(t, bal)
			})
		}
	}
	responses, err := eng.Transfer(pos)
	assert.Equal(t, 2, len(responses))
	assert.NoError(t, err)
	resp := responses[0]
	// total balance of settlement account should be 3 times price
	for _, bal := range resp.Balances {
		if bal.Account.Type == types.AccountType_SETTLEMENT {
			// rounding error -> 1666 + 833 == 2499 assert.Equal(t, int64(1), bal.Account.Balance) }
			assert.Equal(t, int64(1), bal.Account.Balance)
		}
	}
	resp = responses[1]
	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 2, len(resp.Transfers))
}

func testProcessBothProRatedMTM(t *testing.T) {
	market := "BTCtest-market"
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	eng := getTestEngine(t, market, nil)
	defer eng.Finish()

	systemAccs := eng.systemAccs
	traderAccs := getTraderAccounts(trader, market)
	moneyAccs := getTraderAccounts(moneyTrader, market)
	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_MTM_LOSS,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  "BTC",
			},
			Type: types.TransferType_MTM_LOSS,
		},
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_MTM_WIN,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  "BTC",
			},
			Type: types.TransferType_MTM_WIN,
		},
	}
	// The, each time we encounter a trader (ie each position aggregate), we'll attempt to create the account
	eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(len(pos) / 2).DoAndReturn(func(owner, market string) ([]*types.Account, error) {
		isTrader := (owner == trader || owner == moneyTrader)
		assert.True(t, isTrader)
		if owner == trader {
			return traderAccs, nil
		}
		return moneyAccs, nil
	})
	// system accounts
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
			// insurance will be used to settle one sale (size 1, of value price, taken from insurance account)
			sacc.Balance = price / 2
			eng.accounts.EXPECT().GetAccountByID(sacc.Id).Times(1).Return(sacc, nil)
			eng.accounts.EXPECT().UpdateBalance(sacc.Id, gomock.Any()).Times(1).Return(nil)
		case types.AccountType_SETTLEMENT:
			// assign to var so we don't need to repeat this loop for sells
			exp := 2 * price
			exp += price / 2
			eng.accounts.EXPECT().GetAccountByID(sacc.Id).Times(1).Return(sacc, nil)
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, exp).Times(1).Return(nil).Do(func(_ string, inc int64) {
				assert.Equal(t, exp, inc)
				// settle.Balance += inc // this should be happening in the code already, though
			})
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, gomock.Any()).Times(2).Return(nil).Do(func(_ string, inc int64) {
				assert.NotZero(t, inc)
			})
		}
	}
	// now settlement for buys on trader with money:
	for _, acc := range moneyAccs {
		switch acc.Type {
		case types.AccountType_MARGIN:
			acc.Balance += 5 * price
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil)
			eng.accounts.EXPECT().IncrementBalance(acc.Id, -2*price).Times(1).Return(nil)
		case types.AccountType_GENERAL:
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil)
			eng.accounts.EXPECT().IncrementBalance(acc.Id, int64(1666)).Times(1).Return(nil)
		}
	}
	for _, acc := range traderAccs {
		switch acc.Type {
		case types.AccountType_GENERAL:
			eng.accounts.EXPECT().IncrementBalance(acc.Id, int64(833)).Times(1).Return(nil)
			fallthrough
		case types.AccountType_MARGIN:
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil).Do(func(_ string, bal int64) {
				assert.NotZero(t, bal)
			})
			fallthrough
		case types.AccountType_MARKET:
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
		}
	}
	// quickly get the interface mocked for this test
	transfers := getMTMTransfer(pos)
	responses, err := eng.MarkToMarket(transfers)
	assert.Equal(t, 2, len(responses))
	assert.NoError(t, err, "was error")
	resp := responses[0]
	// total balance of settlement account should be 3 times price
	for _, bal := range resp.Balances {
		if bal.Account.Type == types.AccountType_SETTLEMENT {
			// rounding error -> 1666 + 833 == 2499 assert.Equal(t, int64(1), bal.Account.Balance) }
			assert.Equal(t, int64(1), bal.Account.Balance)
		}
	}
	resp = responses[1]
	// there should be 3 ledger moves -> settle to trader 1, settle to trader 2, insurance to trader 2
	assert.Equal(t, 2, len(resp.Transfers))
}
*/

func getTestEngine(t *testing.T, market string, insuranceBalance int64) *testEngine {
	ctrl := gomock.NewController(t)
	buf := mocks.NewMockAccountBuffer(ctrl)
	conf := collateral.NewDefaultConfig()
	buf.EXPECT().Add(gomock.Any()).Times(2)

	eng, err := collateral.New(logging.NewTestLogger(), conf, buf)
	assert.Nil(t, err)

	// create market and traders used for tests
	insID, setID := eng.CreateMarketAccounts(testMarketID, testMarketAsset, insuranceBalance)
	assert.Nil(t, err)

	return &testEngine{
		Engine:             eng,
		ctrl:               ctrl,
		buf:                buf,
		marketInsuranceID:  insID,
		marketSettlementID: setID,
		// systemAccs: accounts,
	}
}

func (te *testEngine) Finish() {
	te.systemAccs = nil
	te.ctrl.Finish()
}

type mtmFake struct {
	t *types.Transfer
}

func (m mtmFake) Party() string             { return "" }
func (m mtmFake) Size() int64               { return 0 }
func (m mtmFake) Price() uint64             { return 0 }
func (m mtmFake) Transfer() *types.Transfer { return m.t }

func getMTMTransfer(transfers []*types.Transfer) []events.Transfer {
	r := make([]events.Transfer, 0, len(transfers))
	for _, t := range transfers {
		r = append(r, &mtmFake{
			t: t,
		})
	}
	return r
}

func getSystemAccounts(market string) []*types.Account {
	return []*types.Account{
		{
			Id:       "system1",
			Owner:    storage.SystemOwner,
			Balance:  0,
			Asset:    market[:3],
			MarketID: market,
			Type:     types.AccountType_SETTLEMENT,
		},
		{
			Id:       "system2",
			Owner:    storage.SystemOwner,
			Balance:  0,
			Asset:    market[:3],
			MarketID: market,
			Type:     types.AccountType_INSURANCE,
		},
	}
}

func getTraderAccounts(trader, market string) []*types.Account {
	return []*types.Account{
		{
			Id:       fmt.Sprintf("%s1", trader),
			Owner:    trader,
			Balance:  0,
			Asset:    market[:3],
			MarketID: market,
			Type:     types.AccountType_MARGIN,
		},
		{
			Id:       fmt.Sprintf("%s2", trader),
			Owner:    trader,
			Balance:  0,
			Asset:    market[:3],
			MarketID: storage.NoMarket,
			Type:     types.AccountType_GENERAL,
		},
	}
}
