package collateral_test

import (
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/internal/events"

	"code.vegaprotocol.io/vega/internal/collateral/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func testTransferChannel(t *testing.T) {
	t.Run("Test channel flow success", testTransferChannelSuccess)
}

// most of this function is copied from the MarkToMarket test - we're using channels, sure
// but the flow should remain the same regardless
func testTransferChannelSuccess(t *testing.T) {
	market := "test-market"
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
	eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(len(pos)).DoAndReturn(func(owner, market string) ([]*types.Account, error) {
		isTrader := (owner == trader || owner == moneyTrader)
		assert.True(t, isTrader)
		if owner == trader {
			return traderAccs, nil
		}
		return moneyAccs, nil
	})
	for _, sacc := range systemAccs {
		switch sacc.Type {
		case types.AccountType_INSURANCE:
			// insurance will be used to settle one sale (size 1, of value price, taken from insurance account)
			sacc.Balance = price / 2
			eng.accounts.EXPECT().GetAccountByID(sacc.Id).Times(1).Return(sacc, nil)
			eng.accounts.EXPECT().UpdateBalance(sacc.Id, int64(0)).Times(1).Return(nil).Do(func(_ string, _ int64) {
				sacc.Balance = 0
			})
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
	// set up getting all trader accounts, ensure balances are set
	for _, acc := range moneyAccs {
		switch acc.Type {
		case types.AccountType_MARGIN:
			acc.Balance += 5 * price
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
			eng.accounts.EXPECT().IncrementBalance(acc.Id, -2*price).Times(1).Return(nil)
		case types.AccountType_MARKET:
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
		case types.AccountType_GENERAL:
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
			eng.accounts.EXPECT().IncrementBalance(acc.Id, int64(1666)).Times(1).Return(nil)
		}
	}
	for _, acc := range traderAccs {
		switch acc.Type {
		case types.AccountType_GENERAL:
			eng.accounts.EXPECT().IncrementBalance(acc.Id, int64(833)).Times(1).Return(nil)
			fallthrough
		case types.AccountType_MARGIN, types.AccountType_MARKET:
			eng.accounts.EXPECT().GetAccountByID(acc.Id).Times(1).Return(acc, nil)
		}
	}
	transfers := eng.getTestMTMTransfer(pos)
	resCh, errCh := eng.TransferCh(transfers)
	responses := make([]events.Margin, 0, len(transfers))
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for res := range resCh {
			responses = append(responses, res)
		}
		wg.Done()
	}()
	wg.Wait()
	assert.Empty(t, errCh)
	assert.Equal(t, 4, len(responses))
}

func (e *testEngine) getTestMTMTransfer(transfers []*types.Transfer) []events.Transfer {
	tt := make([]events.Transfer, 0, len(transfers))
	for _, t := range transfers {
		mt := mocks.NewMockMTMTransfer(e.ctrl)
		mt.EXPECT().Transfer().MinTimes(1).Return(t)
		tt = append(tt, mt)
	}
	return tt
}
