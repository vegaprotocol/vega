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
