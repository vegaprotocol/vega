package collat_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/events"

	"code.vegaprotocol.io/vega/collateral/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestTransferChannel(t *testing.T) {
	// t.Run("Test channel flow success", testTransferChannelSuccess)
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
	mID, gID := eng.Engine.CreateTraderAccount(trader, testMarketID, testMarketAsset)
	assert.NotEmpty(t, mID)
	assert.NotEmpty(t, gID)

	// create + add balance
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

	eng.buf.EXPECT().Add(gomock.Any()).AnyTimes().Do(func(acc types.Account) {
		if acc.Owner == trader && acc.Type == types.AccountType_GENERAL {
			assert.Equal(t, acc.Balance, int64(833))
		}
		if acc.Owner == moneyTrader && acc.Type == types.AccountType_GENERAL {
			assert.Equal(t, acc.Balance, int64(1666))
		}
	})
	transfers := eng.getTestMTMTransfer(pos)
	evts, raw, err := eng.MarkToMarket(testMarketID, transfers)
	fmt.Printf("err: %+v\n", err)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d (%#v)\n", len(raw), raw)
	fmt.Printf("%d (%#v)\n", len(evts), evts)
	assert.NoError(t, err)
	// assert.Equal(t, 4, len(raw))
	assert.NotEmpty(t, evts)
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
