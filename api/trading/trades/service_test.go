package trades

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"vega/datastore"
	"vega/datastore/mocks"
)

func TestNewTradeService(t *testing.T) {
	var newTradeService = NewTradeService()
	assert.NotNil(t, newTradeService)
}

func TestGetTradesOnAllMarkets(t *testing.T) {
	var market = "MKT/A"

	var tradeStore = mocks.TradeStore{}
	var tradeService = NewTradeService()
	tradeService.Init(&tradeStore)
	tradeStore.On("All", market).Return([]*datastore.Trade{
		{ ID: "A", Market: market, Price:1, },
		{ ID: "B", Market: market, Price:2, },
		{ ID: "C", Market: market, Price:3, },
	}, nil).Once()

	var tradeSet, err = tradeService.GetTrades(market)

	assert.Nil(t, err)
	assert.NotNil(t, tradeSet)
	assert.Equal(t, 3, len(tradeSet))
	tradeStore.AssertExpectations(t)
}

func TestGetTradesForOrderOnMarket(t *testing.T) {
	var market = "MKT/A"
	var orderID = "Z"

	var tradeStore = mocks.TradeStore{}
	var tradeService = NewTradeService()
	tradeService.Init(&tradeStore)
	tradeStore.On("FindByOrderID", market, orderID).Return([]*datastore.Trade{
		{ ID: "A", Market: market, Price:1, OrderID: orderID },
		{ ID: "B", Market: market, Price:2, OrderID: orderID },
		{ ID: "C", Market: market, Price:3, OrderID: orderID },
		{ ID: "D", Market: market, Price:4, OrderID: orderID },
		{ ID: "E", Market: market, Price:5, OrderID: orderID },
		{ ID: "F", Market: market, Price:6, OrderID: orderID },
	}, nil).Once()

	var tradeSet, err = tradeService.GetTradesForOrder(market, orderID)

	assert.Nil(t, err)
	assert.NotNil(t, tradeSet)
	assert.Equal(t, 6, len(tradeSet))
	tradeStore.AssertExpectations(t)
}


