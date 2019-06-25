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

func TestTransferChannel(t *testing.T) {
	t.Run("Test channel flow success", testTransferChannelSuccess)
}

// most of this function is copied from the MarkToMarket test - we're using channels, sure
// but the flow should remain the same regardless
func testTransferChannelSuccess(t *testing.T) {
	trader := "test-trader"
	moneyTrader := "money-trader"
	price := int64(1000)

	eng := getTestEngine(t, testMarketID, price/2)
	defer eng.Finish()

	// create trader accounts
	eng.buf.EXPECT().Add(gomock.Any()).Times(2)
	_, _ = eng.Engine.CreateTraderAccount(trader, testMarketID, testMarketAsset)

	eng.buf.EXPECT().Add(gomock.Any()).Times(3)
	marginMoneyTrader, _ := eng.Engine.CreateTraderAccount(moneyTrader, testMarketID, testMarketAsset)
	err := eng.Engine.UpdateBalance(marginMoneyTrader, 5*price)
	assert.Nil(t, err)

	pos := []*types.Transfer{
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_LOSS,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: -price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_LOSS,
		},
		{
			Owner: trader,
			Size:  1,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_WIN,
		},
		{
			Owner: moneyTrader,
			Size:  2,
			Amount: &types.FinancialAmount{
				Amount: price,
				Asset:  testMarketAsset,
			},
			Type: types.TransferType_MTM_WIN,
		},
	}

	/*
		// The, each time we encounter a trader (ie each position aggregate), we'll attempt to create the account
		eng.accounts.EXPECT().CreateTraderMarketAccounts(gomock.Any(), market).Times(len(pos) / 2).DoAndReturn(func(owner, market string) ([]*types.Account, error) {
			isTrader := (owner == trader || owner == moneyTrader)
			assert.True(t, isTrader)
			if owner == trader {
				return traderAccs, nil
			}
			return moneyAccs, nil
		})
	*/

	/*
		buf.EXPECT().Add(gomock.Any()).Times(1)
		err := eng.Engine.UpdateBalance(eng.marketInsuranceID, price/2)

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
				eng.accounts.EXPECT().GetAccountByID(acc.Id).MinTimes(1).MaxTimes(2).Return(acc, nil)
				eng.accounts.EXPECT().UpdateBalance(acc.Id, gomock.Any()).Times(1).Return(nil)
				eng.accounts.EXPECT().IncrementBalance(acc.Id, -2*price).Times(1).Return(nil)
			case types.AccountType_GENERAL:
				eng.accounts.EXPECT().GetAccountByID(acc.Id).MinTimes(1).MaxTimes(2).Return(acc, nil)
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
	*/

	eng.buf.EXPECT().Add(gomock.Any()).Times(7).Do(func(acc types.Account) {
		if acc.Owner == trader && acc.Type == types.AccountType_GENERAL {
			assert.Equal(t, acc.Balance, int64(833))
		}
		if acc.Owner == moneyTrader && acc.Type == types.AccountType_GENERAL {
			assert.Equal(t, acc.Balance, int64(1666))
		}
	})
	transfers := eng.getTestMTMTransfer(pos)
	resCh, errCh := eng.TransferCh(testMarketID, transfers)
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

		mt := mocks.NewMockTransfer(e.ctrl)
		mt.EXPECT().Transfer().AnyTimes().Return(t)
		tt = append(tt, mt)
	}
	return tt
}
