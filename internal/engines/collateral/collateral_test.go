package collateral_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/internal/engines/collateral"
	"code.vegaprotocol.io/vega/internal/engines/collateral/mocks"
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

func TestCollateral(t *testing.T) {
	t.Run("test creating new - should create market accounts", testNew)
	t.Run("test collecting buys - both insurance and sufficient in trader accounts", testCollectBuy)
	t.Run("test collecting buys - trader account not empty, but insufficient", testCollectComplexBuy)
	t.Run("test collecting sells - cases where settle account is full + where insurance pool is tapped", testCollectSell)
}

func testNew(t *testing.T) {
	eng := getTestEngine(t, "test-market", nil)
	eng.ctrl.Finish()
	eng = getTestEngine(t, "test-market", errors.New("random error"))
	eng.ctrl.Finish()
}

func testCollectBuy(t *testing.T) {
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
	pos := []*types.SettlePosition{
		{
			Owner: trader,
			Size:  1,
			Price: price,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Price: price, // should yield -2000 on margin account balance
		},
	}
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
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
	resp, err := eng.CollectBuys(pos)
	assert.NoError(t, err)
	// total balance of settlement account should be 3 times price
	assert.Equal(t, 3*price, resp.Balances[0].Balance)
	// there should be 2 ledger moves
	assert.Equal(t, 2, len(resp.Transfers))
}

func testCollectComplexBuy(t *testing.T) {
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
	pos := []*types.SettlePosition{
		{
			Owner: trader,
			Size:  1,
			Price: price,
		},
	}
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
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
	resp, err := eng.CollectBuys(pos)
	assert.NoError(t, err)
	// total balance should equal price (only 1 call after all)
	assert.Equal(t, price, resp.Balances[0].Balance)
	// there should be 2 ledger moves, one from trader account, one from insurance acc
	assert.Equal(t, 2, len(resp.Transfers))
}

func testCollectSell(t *testing.T) {
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
	pos := []*types.SettlePosition{
		{
			Owner: trader,
			Size:  1,
			Price: price,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Price: price,
		},
	}
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
			// insurance will be used to settle one more sale
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
			eng.accounts.EXPECT().GetAccountsForOwnerByType(trader, tacc.Type).Times(1).Return(tacc, nil)
			break
		}
	}
	for _, tacc := range moneyAccs {
		// ensure trader has money in account
		if tacc.Type == types.AccountType_GENERAL {
			// update balance accordingly
			eng.accounts.EXPECT().GetAccountsForOwnerByType(moneyTrader, tacc.Type).Times(1).Return(tacc, nil)
			eng.accounts.EXPECT().IncrementBalance(tacc.Id, 2*price).Times(1).Return(nil)
			break
		}
	}
	resp, err := eng.CollectSells(pos)
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

func getTestEngine(t *testing.T, market string, err error) *testEngine {
	ctrl := gomock.NewController(t)
	acc := mocks.NewMockAccounts(ctrl)
	conf := collateral.NewDefaultConfig(logging.NewTestLogger())
	acc.EXPECT().CreateMarketAccounts(market, int64(0)).Times(1).Return(err)
	eng, err2 := collateral.New(conf, market, acc)
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

func getSystemAccounts(market string) []*types.Account {
	return []*types.Account{
		{
			Id:      "system1",
			Owner:   storage.SystemOwner,
			Balance: 0,
			Asset:   "",
			Market:  market,
			Type:    types.AccountType_SETTLEMENT,
		},
		{
			Id:      "system2",
			Owner:   storage.SystemOwner,
			Balance: 0,
			Asset:   "",
			Market:  market,
			Type:    types.AccountType_INSURANCE,
		},
	}
}

func getTraderAccounts(trader, market string) []*types.Account {
	return []*types.Account{
		{
			Id:      fmt.Sprintf("%s1", trader),
			Owner:   trader,
			Balance: 0,
			Asset:   "",
			Market:  "",
			Type:    types.AccountType_MARGIN,
		},
		{
			Id:      fmt.Sprintf("%s2", trader),
			Owner:   trader,
			Balance: 0,
			Asset:   "",
			Market:  "",
			Type:    types.AccountType_GENERAL,
		},
		{
			Id:      fmt.Sprintf("%s3", trader),
			Owner:   trader,
			Balance: 0,
			Asset:   "",
			Market:  market,
			Type:    types.AccountType_MARKET,
		},
	}
}
