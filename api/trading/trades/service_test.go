package trades

import (
	"testing"
	"context"
	"github.com/stretchr/testify/assert"
	"vega/datastore"
	"vega/datastore/mocks"
	"vega/proto"
)

func TestNewTradeService(t *testing.T) {
	var newTradeService = NewTradeService()
	assert.NotNil(t, newTradeService)
}

func TestGetTradesOnAllMarkets(t *testing.T) {
	var market = "MKT/A"

	var ctx = context.Background()
	var tradeStore = mocks.TradeStore{}
	var tradeService = NewTradeService()
	tradeService.Init(&tradeStore)
	tradeStore.On("All", ctx, market).Return([]*datastore.Trade{
		{ Trade: msg.Trade { Id: "A", Market: market, Price:1, } },
		{ Trade: msg.Trade { Id: "B", Market: market, Price:2, } },
		{ Trade: msg.Trade { Id: "C", Market: market, Price:3, } },
	}, nil).Once()

	var tradeSet, err = tradeService.GetTrades(ctx, market)

	assert.Nil(t, err)
	assert.NotNil(t, tradeSet)
	assert.Equal(t, 3, len(tradeSet))
	tradeStore.AssertExpectations(t)
}

func TestGetTradesForOrderOnMarket(t *testing.T) {
	var market = "MKT/A"
	var orderID = "Z"

	var ctx = context.Background()
	var tradeStore = mocks.TradeStore{}
	var tradeService = NewTradeService()
	tradeService.Init(&tradeStore)
	tradeStore.On("GetByOrderID", ctx, market, orderID).Return([]*datastore.Trade{
		{ Trade: msg.Trade { Id: "A", Market: market, Price:1 }, OrderID: orderID },
		{ Trade: msg.Trade { Id: "B", Market: market, Price:2 }, OrderID: orderID },
		{ Trade: msg.Trade { Id: "C", Market: market, Price:3 }, OrderID: orderID },
		{ Trade: msg.Trade { Id: "D", Market: market, Price:4 }, OrderID: orderID },
		{ Trade: msg.Trade { Id: "E", Market: market, Price:5 }, OrderID: orderID },
		{ Trade: msg.Trade { Id: "F", Market: market, Price:6 }, OrderID: orderID },
	}, nil).Once()

	var tradeSet, err = tradeService.GetTradesForOrder(ctx, market, orderID)

	assert.Nil(t, err)
	assert.NotNil(t, tradeSet)
	assert.Equal(t, 6, len(tradeSet))
	tradeStore.AssertExpectations(t)
}


