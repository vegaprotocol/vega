package collateral_test

import (
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/internal/engines/collateral/mocks"
	"code.vegaprotocol.io/vega/internal/engines/events"
	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestTransferChannel(t *testing.T) {
	t.Run("Test channel flow success", testTransferChannelSuccess)
}

// most of this function is copied from the MarkToMarket test - we're using channels, sure
// but the flow should remain the same regardless
func testTransferChannelSuccess(t *testing.T) {
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
	// this is different, we're always getting all accounts for traders
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, trader).Times(1).Return(traderAccs, nil)
	eng.accounts.EXPECT().GetMarketAccountsForOwner(market, moneyTrader).Times(1).Return(moneyAccs, nil)
	// now, settle account will be debited per sell position, so 2 calls:
	eng.accounts.EXPECT().IncrementBalance(settle.Id, gomock.Any()).Times(2).Return(nil).Do(func(_ string, inc int64) {
		assert.NotZero(t, inc)
	})
	// next up, updating the balance of the traders' general accounts
	eng.accounts.EXPECT().IncrementBalance(tGeneral.Id, int64(833)).Times(1).Return(nil)
	eng.accounts.EXPECT().IncrementBalance(mGeneral.Id, int64(1666)).Times(1).Return(nil)
	transfers := eng.getTestMTMTransfer(pos)
	resCh, errCh := eng.TransferCh(transfers)
	responses := make([]events.MarginChange, 0, len(transfers))
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

func (e *testEngine) getTestMTMTransfer(transfers []*types.Transfer) []events.MTMTransfer {
	tt := make([]events.MTMTransfer, 0, len(transfers))
	for _, t := range transfers {
		mt := mocks.NewMockMTMTransfer(e.ctrl)
		mt.EXPECT().Transfer().MinTimes(1).Return(t)
		tt = append(tt, mt)
	}
	return tt
}
