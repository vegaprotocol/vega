package api

import (
	"context"
	"testing"
	"time"

	"vega/core"
	"vega/datastore"
	"vega/datastore/mocks"
	"vega/proto"

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

	vega := &core.Vega{}
	tradeService.Init(vega, &tradeStore)

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

	vega := &core.Vega{}
	tradeService.Init(vega, &tradeStore)

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

	vega := &core.Vega{}
	orderService.Init(vega, orderStore)

	orderStore.On("Get", market, orderId).Return(datastore.Order{
		Order: msg.Order{Id: orderId, Market: market},
	}, nil)

	var order, err = orderService.GetById(ctx, market, orderId)

	assert.Nil(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, orderId, order.Id)
	orderStore.AssertExpectations(t)

}

func TestOrderService_GetOrders(t *testing.T) {
	var market = ServiceTestMarket
	var party = ""
	var ctx = context.Background()
	var orderStore = mocks.OrderStore{}
	var orderService = NewOrderService()

	vega := &core.Vega{}
	orderService.Init(vega, &orderStore)

	orderStore.On("GetAll", market, party, datastore.GetParams{Limit: datastore.GetParamsLimitDefault}).Return([]datastore.Order{
		{Order: msg.Order{Id: "A", Market: market, Price: 1, Party: party}},
		{Order: msg.Order{Id: "B", Market: market, Price: 2, Party: party}},
		{Order: msg.Order{Id: "C", Market: market, Price: 3, Party: party}},
		{Order: msg.Order{Id: "D", Market: market, Price: 4, Party: party}},
		{Order: msg.Order{Id: "E", Market: market, Price: 5, Party: party}},
	}, nil).Once()

	var orders, err = orderService.GetOrders(ctx, market, party, datastore.GetParamsLimitDefault)

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

	vega := &core.Vega{}
	tradeService.Init(vega, &tradeStore)
	tradeStore.On("Get", market, tradeId).Return(datastore.Trade{
		Trade: msg.Trade{Id: tradeId, Market: market},
	}, nil)

	var trade, err = tradeService.GetById(ctx, market, tradeId)

	assert.Nil(t, err)
	assert.NotNil(t, trade)
	assert.Equal(t, tradeId, trade.Id)
	tradeStore.AssertExpectations(t)
}

func TestTradeService_GetCandlesChart(t *testing.T) {
	var market = ServiceTestMarket
	const genesisTimeStr = "2018-07-09T12:00:00Z"
	genesisT, _ := time.Parse(time.RFC3339, genesisTimeStr)

	nowT := genesisT.Add(6 * time.Minute)

	// genesis is 6 minutes ago, retrieve information for last 5 minutes and organise it in 1 minute blocks
	// which is interval 60 as there are 60 blocks in 1 minute.
	// This should result in 5 candles

	since := nowT.Add(-5 * time.Minute)
	interval := uint64(60)

	var ctx = context.Background()
	var tradeStore = mocks.TradeStore{}
	var tradeService = NewTradeService()

	vega := &core.Vega{}
	vega.State = &core.State{}
	vega.State.Height = 6 * 60

	tradeService.Init(vega, &tradeStore)
	sinceInBlocks := uint64(60)

	tradeStore.On("GetCandles", market, sinceInBlocks, uint64(vega.State.Height), interval).Return(msg.Candles{
		Candles: []*msg.Candle{
			{High: 112, Low: 109, Open: 110, Close: 112, Volume: 10598},
			{High: 114, Low: 111, Open: 111, Close: 112, Volume: 6360},
			{High: 119, Low: 113, Open: 113, Close: 117, Volume: 17892},
			{High: 117, Low: 116, Open: 116, Close: 116, Volume: 3061},
			{High: 124, Low: 115, Open: 115, Close: 124, Volume: 9613},
		},
	}, nil).Once()

	candles, err := tradeService.GetCandlesChart(ctx, market, since, interval)

	assert.Nil(t, err)
	assert.NotNil(t, candles)
	assert.Equal(t, 5, len(candles.Candles))

	assert.Equal(t, "2018-07-09T12:01:00Z", candles.Candles[0].Date)
	assert.Equal(t, "2018-07-09T12:02:00Z", candles.Candles[1].Date)
	assert.Equal(t, "2018-07-09T12:03:00Z", candles.Candles[2].Date)
	assert.Equal(t, "2018-07-09T12:04:00Z", candles.Candles[3].Date)
	assert.Equal(t, "2018-07-09T12:05:00Z", candles.Candles[4].Date)
}
