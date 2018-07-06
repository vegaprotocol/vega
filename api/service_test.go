package api

import (
	"testing"
	"context"

	"vega/datastore"
	"vega/proto"
	"vega/datastore/mocks"

	"github.com/stretchr/testify/assert"
)

func TestNewTradeService(t *testing.T) {
	var newTradeService = NewTradeService()
	assert.NotNil(t, newTradeService)
}

const ServiceTestMarket = "BTC/DEC18"

func TestTradeService_TestGetAllTradesOnMarket(t *testing.T) {
	var market = ServiceTestMarket

	var ctx = context.Background()
	var tradeStore = mocks.TradeStore{}
	var tradeService = NewTradeService()
	tradeService.Init(&tradeStore)

	tradeStore.On("GetAll", market, datastore.GetParams{Limit: datastore.GetParamsLimitDefault}).Return([]datastore.Trade{
		{Trade: msg.Trade{Id: "A", Market: market, Price: 1}},
		{Trade: msg.Trade{Id: "B", Market: market, Price: 2}},
		{Trade: msg.Trade{Id: "C", Market: market, Price: 3}},
	}, nil).Once()

	var tradeSet, err = tradeService.GetTrades(ctx, market, datastore.GetParamsLimitDefault)

	assert.Nil(t, err)
	assert.NotNil(t, tradeSet)
	assert.Equal(t, 3, len(tradeSet))
	tradeStore.AssertExpectations(t)
}

func TestTradeService_GetAllTradesForOrderOnMarket(t *testing.T) {
	var market = ServiceTestMarket
	var orderId = "12345"

	var ctx = context.Background()
	var tradeStore = mocks.TradeStore{}
	var tradeService = NewTradeService()
	tradeService.Init(&tradeStore)
	tradeStore.On("GetByOrderId", market, orderId, datastore.GetParams{Limit: datastore.GetParamsLimitDefault}).Return([]datastore.Trade{
		{Trade: msg.Trade{Id: "A", Market: market, Price: 1}, OrderId: orderId},
		{Trade: msg.Trade{Id: "B", Market: market, Price: 2}, OrderId: orderId},
		{Trade: msg.Trade{Id: "C", Market: market, Price: 3}, OrderId: orderId},
		{Trade: msg.Trade{Id: "D", Market: market, Price: 4}, OrderId: orderId},
		{Trade: msg.Trade{Id: "E", Market: market, Price: 5}, OrderId: orderId},
		{Trade: msg.Trade{Id: "F", Market: market, Price: 6}, OrderId: orderId},
	}, nil).Once()

	var tradeSet, err = tradeService.GetTradesForOrder(ctx, market, orderId, datastore.GetParamsLimitDefault)

	assert.Nil(t, err)
	assert.NotNil(t, tradeSet)
	assert.Equal(t, 6, len(tradeSet))
	tradeStore.AssertExpectations(t)
}

func TestOrderService_GetOrderById(t *testing.T) {
	var market = ServiceTestMarket
	var orderId = "12345"

	var ctx = context.Background()
	var orderStore = mocks.OrderStore{}
	var orderService = NewOrderService()
	orderService.Init(&orderStore)

	orderStore.On("Get", market, orderId).Return(datastore.Order{
		Order: msg.Order{ Id: orderId, Market: market },
	}, nil)

	var order, err = orderService.GetById(ctx, market, orderId)

	assert.Nil(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, orderId, order.Id)
	orderStore.AssertExpectations(t)
	
}

func TestOrderService_GetOrders(t *testing.T) {
	var market = ServiceTestMarket
	var ctx = context.Background()
	var orderStore = mocks.OrderStore{}
	var orderService = NewOrderService()
	orderService.Init(&orderStore)

	orderStore.On("GetAll", market, "", datastore.GetParams{Limit: datastore.GetParamsLimitDefault}).Return([]datastore.Order{
		{Order: msg.Order{Id: "A", Market: market, Price: 1},},
		{Order: msg.Order{Id: "B", Market: market, Price: 2},},
		{Order: msg.Order{Id: "C", Market: market, Price: 3},},
		{Order: msg.Order{Id: "D", Market: market, Price: 4},},
		{Order: msg.Order{Id: "E", Market: market, Price: 5},},
	}, nil).Once()

	var orders, err = orderService.GetOrders(ctx, market, datastore.GetParamsLimitDefault)

	assert.Nil(t, err)
	assert.NotNil(t, orders)
	assert.Equal(t, 5, len(orders))
	orderStore.AssertExpectations(t)
}

func TestTradeService_GetTradeById(t *testing.T) {
	var market = ServiceTestMarket
	var tradeId = "54321"

	var ctx = context.Background()
	var tradeStore = mocks.TradeStore{}
	var tradeService = NewTradeService()
	tradeService.Init(&tradeStore)
	tradeStore.On("Get", market, tradeId).Return(datastore.Trade{
		Trade: msg.Trade{ Id: tradeId, Market: market },
	}, nil)

	var trade, err = tradeService.GetById(ctx, market, tradeId)

	assert.Nil(t, err)
	assert.NotNil(t, trade)
	assert.Equal(t, tradeId, trade.Id)
	tradeStore.AssertExpectations(t)
}


