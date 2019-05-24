package collateral_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/internal/events"

	"code.vegaprotocol.io/vega/internal/collateral"
	"code.vegaprotocol.io/vega/internal/collateral/mocks"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type testEngine struct {
	*collateral.Engine
	ctrl       *gomock.Controller
	accounts   *mocks.MockAccounts
	systemAccs []*types.Account
}

func TestCollateralTransfer(t *testing.T) {
	t.Run("test creating new - should create market accounts", testNew)
	t.Run("test collecting buys - both insurance and sufficient in trader accounts", testTransferLoss)
	t.Run("test collecting buys - trader account not empty, but insufficient", testTransferComplexLoss)
	t.Run("test collecting buys - trader missing some accounts", testTransferLossMissingTraderAccounts)
	t.Run("test collecting sells - cases where settle account is full + where insurance pool is tapped", testDistributeWin)
	t.Run("test collecting both buys and sells - Successfully collect buy and sell in a single call", testProcessBoth)
	t.Run("test distribution insufficient funds - Transfer losses (partial), distribute wins pro-rate", testProcessBothProRated)
}

func TestCollateralMarkToMarket(t *testing.T) {
	t.Run("Mark to Market distribution, insufficient funcs - complex scenario", testProcessBothProRatedMTM)
}

func TestAddTraderToMarket(t *testing.T) {
	t.Run("Successful calls adding new traders (one duplicate, one actual new)", testAddTrader)
}

func testNew(t *testing.T) {
	eng := getTestEngine(t, "test-market", nil)
	eng.ctrl.Finish()
	eng = getTestEngine(t, "test-market", errors.New("random error"))
	eng.ctrl.Finish()
}

func testAddTrader(t *testing.T) {
	market := "test-market"
	eng := getTestEngine(t, market, nil)
	traders := []string{
		"duplicate",
		"success",
	}
	traderAccs := getTraderAccounts(traders[1], market)
	var gen *types.Account
	for i := range traderAccs {
		if traderAccs[i].Type == types.AccountType_GENERAL {
			gen = traderAccs[i]
			traderAccs = append(traderAccs[:i], traderAccs[i+1:]...)
			break
		}
	}
	assert.NotNil(t, gen)
	// this trader already exists, skip this stuff
	eng.accounts.EXPECT().CreateTraderMarketAccounts(traders[0], market).Times(1).Return(errors.New("already exists"))
	// this trader will be set up successfully
	eng.accounts.EXPECT().CreateTraderMarketAccounts(traders[1], market).Times(1).Return(nil)
	eng.accounts.EXPECT().GetAccountsForOwnerByType(gen.Owner, types.AccountType_GENERAL).Times(1).Return([]*types.Account{gen}, nil)
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, traders[1]).Times(1).Return(traderAccs, nil)
	// expected balances
	general := eng.Config.TraderGeneralAccountBalance
	margin := general / 100 * eng.Config.TraderMarginPercent
	general -= margin
	for _, acc := range traderAccs {
		if acc.Type == types.AccountType_MARGIN || acc.Type == types.AccountType_GENERAL {
			eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil).Do((func(acc *types.Account) func(string, int64) error {
				return func(id string, balance int64) error {
					if acc.Type == types.AccountType_MARGIN {
						assert.Equal(t, margin, balance)
					} else {
						assert.Equal(t, general, balance)
					}
					return nil
				}
			}(acc)))
		}
	}
	eng.accounts.EXPECT().UpdateBalance(gen.Id, general).Times(1).Return(nil)
	for _, trader := range traders {
		assert.NoError(t, eng.AddTraderToMarket(trader))
	}
	eng.ctrl.Finish()
}

func testTransferLoss(t *testing.T) {
	market := "test-market"
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	systemAccs := getSystemAccounts(market)
	traderAccs := getTraderAccounts(trader, market)
	moneyAccs := getTraderAccounts(moneyTrader, market)
	eng := getTestEngine(t, market, nil)
	defer eng.ctrl.Finish()
	// we're going to auto-create the accounts
	eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(2).Return(nil).Do(func(owner, market string) {
		isTrader := (owner == trader || owner == moneyTrader)
		assert.True(t, isTrader)
	})
	// set up the get-accounts calls
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, storage.SystemOwner).Times(1).Return(systemAccs, nil)
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, trader).Times(1).Return(traderAccs, nil)
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, moneyTrader).Times(1).Return(moneyAccs, nil)
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
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
			sacc.Balance = price * 5
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, -price).Times(1).Return(nil)
		case types.AccountType_SETTLEMENT:
			// update settlement account balance
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, 3*price).Times(1).Return(nil)
		}
	}
	for _, tacc := range moneyAccs {
		// ensure trader has money in account
		if tacc.Type == types.AccountType_MARGIN {
			tacc.Balance += 100000
			// update balance accordingly
			eng.accounts.EXPECT().IncrementBalance(tacc.Id, -2*price).Times(1).Return(nil)
		}
	}
	responses, err := eng.Transfer(pos)
	assert.Equal(t, 1, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance of settlement account should be 3 times price
	assert.Equal(t, 3*price, resp.Balances[0].Balance)
	// there should be 2 ledger moves
	assert.Equal(t, 2, len(resp.Transfers))
}

func testTransferComplexLoss(t *testing.T) {
	market := "test-market"
	trader := "test-trader"
	half := int64(500)
	price := half * 2

	systemAccs := getSystemAccounts(market)
	traderAccs := getTraderAccounts(trader, market)
	eng := getTestEngine(t, market, nil)
	defer eng.ctrl.Finish()
	// we're going to auto-create the accounts
	eng.accounts.EXPECT().CreateTraderMarketAccounts(trader, market).Times(1).Return(nil)
	// set up the get-accounts calls
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, storage.SystemOwner).Times(1).Return(systemAccs, nil)
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, trader).Times(1).Return(traderAccs, nil)
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
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
			sacc.Balance = half
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, -half).Times(1).Return(nil)
		case types.AccountType_SETTLEMENT:
			// update settlement account balance
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, price).Times(1).Return(nil)
		}
	}
	for _, tacc := range traderAccs {
		// ensure trader has money in account
		if tacc.Type == types.AccountType_MARGIN {
			tacc.Balance += half
			// update balance accordingly
			eng.accounts.EXPECT().UpdateBalance(tacc.Id, int64(0)).Times(1).Return(nil)
		}
	}
	responses, err := eng.Transfer(pos)
	assert.Equal(t, 1, len(responses))
	resp := responses[0]
	assert.NoError(t, err)
	// total balance should equal price (only 1 call after all)
	assert.Equal(t, price, resp.Balances[0].Balance)
	// there should be 2 ledger moves, one from trader account, one from insurance acc
	assert.Equal(t, 2, len(resp.Transfers))
}

func testTransferLossMissingTraderAccounts(t *testing.T) {
	market := "test-market"
	trader := "test-trader"
	price := int64(1000)

	systemAccs := getSystemAccounts(market)
	allAccs := getTraderAccounts(trader, market)
	traderAccs := make([]*types.Account, 0, len(allAccs)-1)
	for _, acc := range allAccs {
		// all but margin account
		if acc.Type != types.AccountType_MARGIN {
			traderAccs = append(traderAccs, acc)
		}
	}
	eng := getTestEngine(t, market, nil)
	defer eng.ctrl.Finish()
	// we're going to auto-create the accounts
	eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(1).Return(nil).Do(func(owner, market string) {
		assert.Equal(t, trader, owner)
	})
	// set up the get-accounts calls
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, storage.SystemOwner).Times(1).Return(systemAccs, nil)
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, trader).Times(1).Return(traderAccs, nil)
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
	market := "test-market"
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	systemAccs := getSystemAccounts(market)
	traderAccs := getTraderAccounts(trader, market)
	moneyAccs := getTraderAccounts(moneyTrader, market)
	eng := getTestEngine(t, market, nil)
	defer eng.ctrl.Finish()
	// we're going to auto-create the accounts
	eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(2).Return(nil).Do(func(owner, market string) {
		isTrader := (owner == trader || owner == moneyTrader)
		assert.True(t, isTrader)
	})
	// set up the get-accounts calls
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, storage.SystemOwner).Times(1).Return(systemAccs, nil)
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
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
			// insurance will be used to settle one more sale
			sacc.Balance = price
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, -price).Times(1).Return(nil)
		case types.AccountType_SETTLEMENT:
			sacc.Balance = 2 * price
			// first increment by taking out the full price
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, -price).Times(1).Return(nil)
			// second time, it's not going to be enough
			eng.accounts.EXPECT().UpdateBalance(sacc.Id, int64(0)).Times(1).Return(nil)
		}
	}
	for _, tacc := range traderAccs {
		if tacc.Type == types.AccountType_GENERAL {
			eng.accounts.EXPECT().IncrementBalance(tacc.Id, price).Times(1).Return(nil)
			eng.accounts.EXPECT().GetAccountsForOwnerByType(trader, tacc.Type).Times(1).Return([]*types.Account{tacc}, nil)
			break
		}
	}
	for _, tacc := range moneyAccs {
		// ensure trader has money in account
		if tacc.Type == types.AccountType_GENERAL {
			// update balance accordingly
			eng.accounts.EXPECT().GetAccountsForOwnerByType(moneyTrader, tacc.Type).Times(1).Return([]*types.Account{tacc}, nil)
			eng.accounts.EXPECT().IncrementBalance(tacc.Id, 2*price).Times(1).Return(nil)
			break
		}
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
	market := "test-market"
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	systemAccs := getSystemAccounts(market)
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
	eng := getTestEngine(t, market, nil)
	defer eng.ctrl.Finish()
	// first up, we'll get the system accounts for the market
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, storage.SystemOwner).Times(1).Return(systemAccs, nil)
	// The, each time we encounter a trader (ie each position aggregate), we'll attempt to create the account
	eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(len(pos)).Return(nil).Do(func(owner, market string) {
		isTrader := (owner == trader || owner == moneyTrader)
		assert.True(t, isTrader)
	})
	// next up, calls to buy positions, get market accounts for owner (for this market)
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, trader).Times(1).Return(traderAccs, nil)
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, moneyTrader).Times(1).Return(moneyAccs, nil)
	// now the positions, calls we expect to be made when processing buys
	// system accounts
	var settle *types.Account
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
			// ensure there's money here
			sacc.Balance = 3 * price
			// insurance will be used to settle one sale (size 1, of value price, taken from insurance account)
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, -price).Times(1).Return(nil)
		case types.AccountType_SETTLEMENT:
			// assign to var so we don't need to repeat this loop for sells
			settle = sacc
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, 3*price).Times(1).Return(nil).Do(func(_ string, inc int64) {
				assert.Equal(t, 3*price, inc)
				// settle.Balance += inc // this should be happening in the code already, though
			})
		}
	}
	// ensure this is set
	assert.NotNil(t, settle)
	// now settlement for buys on trader with money:
	for _, tacc := range moneyAccs {
		if tacc.Type == types.AccountType_MARGIN {
			tacc.Balance += 5 * price // plenty
			// we expect the settle of size 2 to be taken from this account
			eng.accounts.EXPECT().IncrementBalance(tacc.Id, -2*price).Times(1).Return(nil)
			break
		}
	}
	// buys should be handled at this point, moving on to sells
	// first thing that'll happen here is getting the general accounts
	var tGeneral, mGeneral *types.Account
	for _, acc := range traderAccs {
		if acc.Type == types.AccountType_GENERAL {
			tGeneral = acc
			break
		}
	}
	// ensure we have this account
	assert.NotNil(t, tGeneral)
	for _, acc := range moneyAccs {
		if acc.Type == types.AccountType_GENERAL {
			mGeneral = acc
			break
		}
	}
	// ensure we have this account
	assert.NotNil(t, mGeneral)
	eng.accounts.EXPECT().GetAccountsForOwnerByType(trader, types.AccountType_GENERAL).Times(1).Return([]*types.Account{tGeneral}, nil)
	eng.accounts.EXPECT().GetAccountsForOwnerByType(moneyTrader, types.AccountType_GENERAL).Times(1).Return([]*types.Account{mGeneral}, nil)
	// now, settle account will be debited per sell position, so 2 calls:
	eng.accounts.EXPECT().IncrementBalance(settle.Id, gomock.Any()).Times(2).Return(nil).Do(func(_ string, inc int64) {
		assert.NotZero(t, inc)
	})
	// next up, updating the balance of the traders' general accounts
	eng.accounts.EXPECT().IncrementBalance(tGeneral.Id, price).Times(1).Return(nil)
	eng.accounts.EXPECT().IncrementBalance(mGeneral.Id, 2*price).Times(1).Return(nil)
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
	market := "test-market"
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	systemAccs := getSystemAccounts(market)
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
	eng := getTestEngine(t, market, nil)
	defer eng.ctrl.Finish()
	// first up, we'll get the system accounts for the market
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, storage.SystemOwner).Times(1).Return(systemAccs, nil)
	// The, each time we encounter a trader (ie each position aggregate), we'll attempt to create the account
	eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(len(pos)).Return(nil).Do(func(owner, market string) {
		isTrader := (owner == trader || owner == moneyTrader)
		assert.True(t, isTrader)
	})
	// next up, calls to buy positions, get market accounts for owner (for this market)
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, trader).Times(1).Return(traderAccs, nil)
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, moneyTrader).Times(1).Return(moneyAccs, nil)
	// now the positions, calls we expect to be made when processing buys
	// system accounts
	var settle *types.Account
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
			// insurance will be used to settle one sale (size 1, of value price, taken from insurance account)
			sacc.Balance = price / 2
			eng.accounts.EXPECT().UpdateBalance(sacc.Id, int64(0)).Times(1).Return(nil).Do(func(_ string, _ int64) {
				sacc.Balance = 0
			})
		case types.AccountType_SETTLEMENT:
			// assign to var so we don't need to repeat this loop for sells
			settle = sacc
			exp := 2 * price
			exp += price / 2
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, exp).Times(1).Return(nil).Do(func(_ string, inc int64) {
				assert.Equal(t, exp, inc)
				// settle.Balance += inc // this should be happening in the code already, though
			})
		}
	}
	// ensure this is set
	assert.NotNil(t, settle)
	// now settlement for buys on trader with money:
	for _, tacc := range moneyAccs {
		if tacc.Type == types.AccountType_MARGIN {
			tacc.Balance += 5 * price // plenty
			// we expect the settle of size 2 to be taken from this account
			eng.accounts.EXPECT().IncrementBalance(tacc.Id, -2*price).Times(1).Return(nil)
			break
		}
	}
	// buys should be handled at this point, moving on to sells
	// first thing that'll happen here is getting the general accounts
	var tGeneral, mGeneral *types.Account
	for _, acc := range traderAccs {
		if acc.Type == types.AccountType_GENERAL {
			tGeneral = acc
			break
		}
	}
	// ensure we have this account
	assert.NotNil(t, tGeneral)
	for _, acc := range moneyAccs {
		if acc.Type == types.AccountType_GENERAL {
			mGeneral = acc
			break
		}
	}
	// ensure we have this account
	assert.NotNil(t, mGeneral)
	eng.accounts.EXPECT().GetAccountsForOwnerByType(trader, types.AccountType_GENERAL).Times(1).Return([]*types.Account{tGeneral}, nil)
	eng.accounts.EXPECT().GetAccountsForOwnerByType(moneyTrader, types.AccountType_GENERAL).Times(1).Return([]*types.Account{mGeneral}, nil)
	// now, settle account will be debited per sell position, so 2 calls:
	eng.accounts.EXPECT().IncrementBalance(settle.Id, gomock.Any()).Times(2).Return(nil).Do(func(_ string, inc int64) {
		assert.NotZero(t, inc)
	})
	// next up, updating the balance of the traders' general accounts
	eng.accounts.EXPECT().IncrementBalance(tGeneral.Id, int64(833)).Times(1).Return(nil)
	eng.accounts.EXPECT().IncrementBalance(mGeneral.Id, int64(1666)).Times(1).Return(nil)
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
	market := "test-market"
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	systemAccs := getSystemAccounts(market)
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
	eng := getTestEngine(t, market, nil)
	defer eng.ctrl.Finish()
	// first up, we'll get the system accounts for the market
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, storage.SystemOwner).Times(1).Return(systemAccs, nil)
	// The, each time we encounter a trader (ie each position aggregate), we'll attempt to create the account
	eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(len(pos)).Return(nil).Do(func(owner, market string) {
		isTrader := (owner == trader || owner == moneyTrader)
		assert.True(t, isTrader)
	})
	// next up, calls to buy positions, get market accounts for owner (for this market)
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, trader).Times(1).Return(traderAccs, nil)
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, moneyTrader).Times(1).Return(moneyAccs, nil)
	// now the positions, calls we expect to be made when processing buys
	// system accounts
	var settle *types.Account
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
			// insurance will be used to settle one sale (size 1, of value price, taken from insurance account)
			sacc.Balance = price / 2
			eng.accounts.EXPECT().UpdateBalance(sacc.Id, int64(0)).Times(1).Return(nil).Do(func(_ string, _ int64) {
				sacc.Balance = 0
			})
		case types.AccountType_SETTLEMENT:
			// assign to var so we don't need to repeat this loop for sells
			settle = sacc
			exp := 2 * price
			exp += price / 2
			eng.accounts.EXPECT().IncrementBalance(sacc.Id, exp).Times(1).Return(nil).Do(func(_ string, inc int64) {
				assert.Equal(t, exp, inc)
				// settle.Balance += inc // this should be happening in the code already, though
			})
		}
	}
	// ensure this is set
	assert.NotNil(t, settle)
	// now settlement for buys on trader with money:
	for _, tacc := range moneyAccs {
		if tacc.Type == types.AccountType_MARGIN {
			tacc.Balance += 5 * price // plenty
			// we expect the settle of size 2 to be taken from this account
			eng.accounts.EXPECT().IncrementBalance(tacc.Id, -2*price).Times(1).Return(nil)
			break
		}
	}
	// buys should be handled at this point, moving on to sells
	// first thing that'll happen here is getting the general accounts
	var tGeneral, mGeneral *types.Account
	for _, acc := range traderAccs {
		if acc.Type == types.AccountType_GENERAL {
			tGeneral = acc
			break
		}
	}
	// ensure we have this account
	assert.NotNil(t, tGeneral)
	for _, acc := range moneyAccs {
		if acc.Type == types.AccountType_GENERAL {
			mGeneral = acc
			break
		}
	}
	// ensure we have this account
	assert.NotNil(t, mGeneral)
	eng.accounts.EXPECT().GetAccountsForOwnerByType(trader, types.AccountType_GENERAL).Times(1).Return([]*types.Account{tGeneral}, nil)
	eng.accounts.EXPECT().GetAccountsForOwnerByType(moneyTrader, types.AccountType_GENERAL).Times(1).Return([]*types.Account{mGeneral}, nil)
	// now, settle account will be debited per sell position, so 2 calls:
	eng.accounts.EXPECT().IncrementBalance(settle.Id, gomock.Any()).Times(2).Return(nil).Do(func(_ string, inc int64) {
		assert.NotZero(t, inc)
	})
	// next up, updating the balance of the traders' general accounts
	eng.accounts.EXPECT().IncrementBalance(tGeneral.Id, int64(833)).Times(1).Return(nil)
	eng.accounts.EXPECT().IncrementBalance(mGeneral.Id, int64(1666)).Times(1).Return(nil)
	// quickly get the interface mocked for this test
	transfers := getMTMTransfer(pos)
	responses, err := eng.MarkToMarket(transfers)
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

func getTestEngine(t *testing.T, market string, err error) *testEngine {
	ctrl := gomock.NewController(t)
	acc := mocks.NewMockAccounts(ctrl)
	conf := collateral.NewDefaultConfig()
	acc.EXPECT().CreateMarketAccounts(market, int64(0)).Times(1).Return(err)
	eng, err2 := collateral.New(logging.NewTestLogger(), conf, market, acc)
	assert.Equal(t, err, err2)
	if err != nil {
		assert.Nil(t, eng)
	}
	return &testEngine{
		Engine:   eng,
		ctrl:     ctrl,
		accounts: acc,
	}
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
			Asset:    "",
			MarketID: market,
			Type:     types.AccountType_SETTLEMENT,
		},
		{
			Id:       "system2",
			Owner:    storage.SystemOwner,
			Balance:  0,
			Asset:    "",
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
			Asset:    "",
			MarketID: "",
			Type:     types.AccountType_MARGIN,
		},
		{
			Id:       fmt.Sprintf("%s2", trader),
			Owner:    trader,
			Balance:  0,
			Asset:    "",
			MarketID: "",
			Type:     types.AccountType_GENERAL,
		},
		{
			Id:       fmt.Sprintf("%s3", trader),
			Owner:    trader,
			Balance:  0,
			Asset:    "",
			MarketID: market,
			Type:     types.AccountType_MARKET,
		},
	}
}
