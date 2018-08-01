package api

import (
	"fmt"
	"math/rand"
	"testing"

	"vega/msg"
	"vega/datastore"

	"github.com/stretchr/testify/assert"
	"context"
	"vega/core"
	"vega/risk"
)

func TestNewTradeService(t *testing.T) {
	var newTradeService = NewTradeService()
	assert.NotNil(t, newTradeService)
}

//const ServiceTestMarket = "BTC/DEC18"

//func TestTradeService_TestGetAllTradesOnMarket(t *testing.T) {
//	var market = ServiceTestMarket
//
//	var ctx = context.Background()
//	var tradeStore = mocks.TradeStore{}
//	var tradeService = NewTradeService()
//
//	vega := &core.Vega{}
//	tradeService.Init(vega, &tradeStore)
//
//	tradeStore.On("GetAll", market, datastore.GetParams{Limit: datastore.GetParamsLimitDefault}).Return([]datastore.Trade{
//		{Trade: msg.Trade{Id: "A", Market: market, Price: 1}},
//		{Trade: msg.Trade{Id: "B", Market: market, Price: 2}},
//		{Trade: msg.Trade{Id: "C", Market: market, Price: 3}},
//	}, nil).Once()
//
//	var tradeSet, err = tradeService.GetTrades(ctx, market, datastore.GetParamsLimitDefault)
//
//	assert.Nil(t, err)
//	assert.NotNil(t, tradeSet)
//	assert.Equal(t, 3, len(tradeSet))
//	tradeStore.AssertExpectations(t)
//}
//
//func TestTradeService_GetAllTradesForOrderOnMarket(t *testing.T) {
//	var market = ServiceTestMarket
//	var orderId = "12345"
//
//	var ctx = context.Background()
//	var tradeStore = mocks.TradeStore{}
//	var tradeService = NewTradeService()
//
//	vega := &core.Vega{}
//	tradeService.Init(vega, &tradeStore)
//
//	tradeStore.On("GetByOrderId", market, orderId, datastore.GetParams{Limit: datastore.GetParamsLimitDefault}).Return([]datastore.Trade{
//		{Trade: msg.Trade{Id: "A", Market: market, Price: 1}, OrderId: orderId},
//		{Trade: msg.Trade{Id: "B", Market: market, Price: 2}, OrderId: orderId},
//		{Trade: msg.Trade{Id: "C", Market: market, Price: 3}, OrderId: orderId},
//		{Trade: msg.Trade{Id: "D", Market: market, Price: 4}, OrderId: orderId},
//		{Trade: msg.Trade{Id: "E", Market: market, Price: 5}, OrderId: orderId},
//		{Trade: msg.Trade{Id: "F", Market: market, Price: 6}, OrderId: orderId},
//	}, nil).Once()
//
//	var tradeSet, err = tradeService.GetTradesForOrder(ctx, market, orderId, datastore.GetParamsLimitDefault)
//
//	assert.Nil(t, err)
//	assert.NotNil(t, tradeSet)
//	assert.Equal(t, 6, len(tradeSet))
//	tradeStore.AssertExpectations(t)
//}
//
//func TestOrderService_GetOrderById(t *testing.T) {
//	var market = ServiceTestMarket
//	var orderId = "12345"
//
//	var ctx = context.Background()
//	var orderStore = mocks.OrderStore{}
//	var orderService = NewOrderService()
//
//	vega := &core.Vega{}
//	orderService.Init(vega, &orderStore)
//
//	orderStore.On("Get", market, orderId).Return(datastore.Order{
//		Order: msg.Order{Id: orderId, Market: market},
//	}, nil)
//
//	var order, err = orderService.GetById(ctx, market, orderId)
//
//	assert.Nil(t, err)
//	assert.NotNil(t, order)
//	assert.Equal(t, orderId, order.Id)
//	orderStore.AssertExpectations(t)
//
//}
//
//func TestOrderService_GetOrders(t *testing.T) {
//	var market = ServiceTestMarket
//	var party = ""
//	var ctx = context.Background()
//	var orderStore = mocks.OrderStore{}
//	var orderService = NewOrderService()
//
//	vega := &core.Vega{}
//	orderService.Init(vega, &orderStore)
//
//	orderStore.On("GetAll", market, party, datastore.GetParams{Limit: datastore.GetParamsLimitDefault}).Return([]datastore.Order{
//		{Order: msg.Order{Id: "A", Market: market, Price: 1, Party: party}},
//		{Order: msg.Order{Id: "B", Market: market, Price: 2, Party: party}},
//		{Order: msg.Order{Id: "C", Market: market, Price: 3, Party: party}},
//		{Order: msg.Order{Id: "D", Market: market, Price: 4, Party: party}},
//		{Order: msg.Order{Id: "E", Market: market, Price: 5, Party: party}},
//	}, nil).Once()
//
//	var orders, err = orderService.GetOrders(ctx, market, party, datastore.GetParamsLimitDefault)
//
//	assert.Nil(t, err)
//	assert.NotNil(t, orders)
//	assert.Equal(t, 5, len(orders))
//	orderStore.AssertExpectations(t)
//}
//
//func TestTradeService_GetTradeById(t *testing.T) {
//	var market = ServiceTestMarket
//	var tradeId = "54321"
//
//	var ctx = context.Background()
//	var tradeStore = mocks.TradeStore{}
//	var tradeService = NewTradeService()
//
//	vega := &core.Vega{}
//	tradeService.Init(vega, &tradeStore)
//	tradeStore.On("Get", market, tradeId).Return(datastore.Trade{
//		Trade: msg.Trade{Id: tradeId, Market: market},
//	}, nil)
//
//	var trade, err = tradeService.GetById(ctx, market, tradeId)
//
//	assert.Nil(t, err)
//	assert.NotNil(t, trade)
//	assert.Equal(t, tradeId, trade.Id)
//	tradeStore.AssertExpectations(t)
//}
//
//func TestTradeService_GetCandlesChart(t *testing.T) {
//	var market = ServiceTestMarket
//	const genesisTimeStr = "2018-07-09T12:00:00Z"
//	genesisT, _ := time.Parse(time.RFC3339, genesisTimeStr)
//
//	nowT := genesisT.Add(6 * time.Minute)
//
//	// genesis is 6 minutes ago, retrieve information for last 5 minutes and organise it in 1 minute blocks
//	// which is interval 60 as there are 60 blocks in 1 minute.
//	// This should result in 5 candles
//
//	since := nowT.Add(-5 * time.Minute)
//	interval := uint64(60)
//
//	var ctx = context.Background()
//	var tradeStore = mocks.TradeStore{}
//	var tradeService = NewTradeService()
//
//	vega := &core.Vega{}
//	vega.State = &core.State{}
//	vega.State.Height = 6 * 60
//
//	tradeService.Init(vega, &tradeStore)
//	sinceInBlocks := uint64(60)
//
//	tradeStore.On("GetCandles", market, sinceInBlocks, uint64(vega.State.Height), interval).Return(msg.Candles{
//		Candles: []*msg.Candle{
//			{High: 112, Low: 109, Open: 110, Close: 112, Volume: 10598},
//			{High: 114, Low: 111, Open: 111, Close: 112, Volume: 6360},
//			{High: 119, Low: 113, Open: 113, Close: 117, Volume: 17892},
//			{High: 117, Low: 116, Open: 116, Close: 116, Volume: 3061},
//			{High: 124, Low: 115, Open: 115, Close: 124, Volume: 9613},
//		},
//	}, nil).Once()
//
//	candles, err := tradeService.GetCandlesChart(ctx, market, since, interval)
//
//	assert.Nil(t, err)
//	assert.NotNil(t, candles)
//	assert.Equal(t, 5, len(candles.Candles))
//
//	assert.Equal(t, "2018-07-09T12:01:00Z", candles.Candles[0].Date)
//	assert.Equal(t, "2018-07-09T12:02:00Z", candles.Candles[1].Date)
//	assert.Equal(t, "2018-07-09T12:03:00Z", candles.Candles[2].Date)
//	assert.Equal(t, "2018-07-09T12:04:00Z", candles.Candles[3].Date)
//	assert.Equal(t, "2018-07-09T12:05:00Z", candles.Candles[4].Date)
//}


func TestTradeService_GetPositionsByParty(t *testing.T) {

}

func TestPositions(t *testing.T) {
	testMarket := "BTC/DEC18"
	testParty := "testParty"
	testPartyA := "testPartyA"
	testPartyB := "testPartyB"

	var memStore = datastore.NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
	var newOrderStore = datastore.NewOrderStore(&memStore)
	var newTradeStore = datastore.NewTradeStore(&memStore)

	var ctx = context.Background()
	var tradeService = NewTradeService()

	vega := &core.Vega{}
	riskEngine := risk.New()
	riskEngine.AddNewMarket(&msg.Market{Name: testMarket})

	tradeService.Init(vega, newTradeStore, riskEngine)

	passiveOrderId := fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrderId := fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId := fmt.Sprintf("%d", rand.Intn(1000000000000))

	aggressiveOrder := &datastore.Order{
		Order: msg.Order{
			Id:     aggressiveOrderId,
			Market: testMarket,
			Party:  testPartyA,
			Side:   msg.Side_Buy,
		},
	}
	passiveOrder := &datastore.Order{
		Order: msg.Order{
			Id:     passiveOrderId,
			Market: testMarket,
			Party:  testPartyB,
			Side:   msg.Side_Sell,
		},
	}

	trade := &datastore.Trade{
		Trade: msg.Trade{
			Id:        tradeId,
			Price:     100,
			Market:    testMarket,
			Size:      500,
			Timestamp: 0,
			Buyer:     testPartyA,
			Seller:    testPartyB,
			Aggressor: msg.Side_Buy,
		},
		PassiveOrderId:    passiveOrderId,
		AggressiveOrderId: aggressiveOrderId,
	}

	err := newOrderStore.Post(*passiveOrder)
	assert.Nil(t, err)
	err = newOrderStore.Post(*aggressiveOrder)
	assert.Nil(t, err)
	err = newTradeStore.Post(*trade)
	assert.Nil(t, err)

	passiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId = fmt.Sprintf("%d", rand.Intn(1000000000000))

	aggressiveOrder = &datastore.Order{
		Order: msg.Order{
			Id:     aggressiveOrderId,
			Market: testMarket,
			Party:  testPartyA,
			Side:   msg.Side_Buy,
		},
	}
	passiveOrder = &datastore.Order{
		Order: msg.Order{
			Id:     passiveOrderId,
			Market: testMarket,
			Party:  testPartyB,
			Side:   msg.Side_Sell,
		},
	}

	trade = &datastore.Trade{
		Trade: msg.Trade{
			Id:        tradeId,
			Price:     100,
			Market:    testMarket,
			Size:      500,
			Timestamp: 0,
			Buyer:     testPartyA,
			Seller:    testPartyB,
			Aggressor: msg.Side_Buy,
		},
		PassiveOrderId:    passiveOrderId,
		AggressiveOrderId: aggressiveOrderId,
	}

	err = newOrderStore.Post(*passiveOrder)
	assert.Nil(t, err)
	err = newOrderStore.Post(*aggressiveOrder)
	assert.Nil(t, err)
	err = newTradeStore.Post(*trade)
	assert.Nil(t, err)


	// two trades of 500 contracts done at the same price of 100
	positions, err := tradeService.GetPositionsByParty(ctx, testPartyA)
	assert.Nil(t, err)

	fmt.Printf("positions returned:\n")
	for _, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, val.Market)
		assert.Equal(t, int64(0), val.RealisedVolume)
		assert.Equal(t, int64(0), val.RealisedPNL)
		assert.Equal(t, int64(1000), val.UnrealisedVolume)
		assert.Equal(t, int64(0), val.UnrealisedPNL)
		assert.Equal(t, int64(550), val.MinimumMargin)
	}

	positions, err = tradeService.GetPositionsByParty(ctx, testPartyB)
	assert.Nil(t, err)

	fmt.Printf("positions returned:\n")
	for _, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(0), val.RealisedVolume)
		assert.Equal(t, int64(0), val.RealisedPNL)
		assert.Equal(t, int64(-1000), val.UnrealisedVolume)
		assert.Equal(t, int64(0), val.UnrealisedPNL)
		assert.Equal(t, int64(553), val.MinimumMargin)
	}

	// market moves by 10 up what is the PNL?
	passiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrder = &datastore.Order{
		Order: msg.Order{
			Id:     aggressiveOrderId,
			Market: testMarket,
			Party:  "partyC",
			Side:   msg.Side_Buy,
		},
	}
	passiveOrder = &datastore.Order{
		Order: msg.Order{
			Id:     passiveOrderId,
			Market: testMarket,
			Party:  "partyD",
			Side:   msg.Side_Sell,
		},
	}

	trade = &datastore.Trade{
		Trade: msg.Trade{
			Id:        tradeId,
			Price:     110,
			Market:    testMarket,
			Size:      1,
			Timestamp: 0,
			Buyer:     "partyC",
			Seller:    "partyD",
			Aggressor: msg.Side_Buy,
		},
		PassiveOrderId:    passiveOrderId,
		AggressiveOrderId: aggressiveOrderId,
	}

	err = newOrderStore.Post(*passiveOrder)
	assert.Nil(t, err)
	err = newOrderStore.Post(*aggressiveOrder)
	assert.Nil(t, err)
	err = newTradeStore.Post(*trade)
	assert.Nil(t, err)

	// current mark price for testMarket should be 110, the PNL for partyA and partyB should change
	markPrice, err := newTradeStore.GetMarkPrice(testMarket)
	assert.Equal(t, uint64(110), markPrice)
	assert.Nil(t, err)

	positions, err = tradeService.GetPositionsByParty(ctx, testPartyA)
	assert.Nil(t, err)

	fmt.Printf("positions returned:\n")
	for _, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(0), val.RealisedVolume)
		assert.Equal(t, int64(0), val.RealisedPNL)
		assert.Equal(t, int64(1000), val.UnrealisedVolume)
		assert.Equal(t, int64(10*1000), val.UnrealisedPNL)
		assert.Equal(t, int64(-9395), val.MinimumMargin)
	}

	positions, err = tradeService.GetPositionsByParty(ctx, testPartyB)
	assert.Nil(t, err)

	fmt.Printf("positions returned:\n")
	for _, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(0), val.RealisedVolume)
		assert.Equal(t, int64(0), val.RealisedPNL)
		assert.Equal(t, int64(-1000), val.UnrealisedVolume)
		assert.Equal(t, int64(-10*1000), val.UnrealisedPNL)
		assert.Equal(t, int64(10608), val.MinimumMargin)
	}


	// close 90% of position at 110
	passiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrder = &datastore.Order{
		Order: msg.Order{
			Id:     aggressiveOrderId,
			Market: testMarket,
			Party:  testPartyA,
			Side:   msg.Side_Sell,
		},
	}
	passiveOrder = &datastore.Order{
		Order: msg.Order{
			Id:     passiveOrderId,
			Market: testMarket,
			Party:  testPartyB,
			Side:   msg.Side_Buy,
		},
	}

	trade = &datastore.Trade{
		Trade: msg.Trade{
			Id:        tradeId,
			Price:     110,
			Market:    testMarket,
			Size:      900,
			Timestamp: 0,
			Buyer:     testPartyB,
			Seller:    testPartyA,
			Aggressor: msg.Side_Sell,
		},
		PassiveOrderId:    passiveOrderId,
		AggressiveOrderId: aggressiveOrderId,
	}

	err = newOrderStore.Post(*passiveOrder)
	assert.Nil(t, err)
	err = newOrderStore.Post(*aggressiveOrder)
	assert.Nil(t, err)
	err = newTradeStore.Post(*trade)
	assert.Nil(t, err)

	positions, err = tradeService.GetPositionsByParty(ctx, testPartyA)
	assert.Nil(t, err)

	fmt.Printf("positions returned:\n")
	for _, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(900), val.RealisedVolume)
		assert.Equal(t, int64(9000), val.RealisedPNL)
		assert.Equal(t, int64(100), val.UnrealisedVolume)
		assert.Equal(t, int64(10*100), val.UnrealisedPNL)
		assert.Equal(t, int64(-940), val.MinimumMargin)
	}

	positions, err = tradeService.GetPositionsByParty(ctx, testPartyB)
	assert.Nil(t, err)

	fmt.Printf("positions returned:\n")
	for _, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(900), val.RealisedVolume)
		assert.Equal(t, int64(-9000), val.RealisedPNL)
		assert.Equal(t, int64(-100), val.UnrealisedVolume)
		assert.Equal(t, int64(-10*100), val.UnrealisedPNL)
		assert.Equal(t, int64(1060), val.MinimumMargin)
	}

	// close remaining 10% of position at 110
	passiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrder = &datastore.Order{
		Order: msg.Order{
			Id:     aggressiveOrderId,
			Market: testMarket,
			Party:  testPartyA,
			Side:   msg.Side_Sell,
		},
	}
	passiveOrder = &datastore.Order{
		Order: msg.Order{
			Id:     passiveOrderId,
			Market: testMarket,
			Party:  testPartyB,
			Side:   msg.Side_Buy,
		},
	}

	trade = &datastore.Trade{
		Trade: msg.Trade{
			Id:        tradeId,
			Price:     110,
			Market:    testMarket,
			Size:      100,
			Timestamp: 0,
			Buyer:     testPartyB,
			Seller:    testPartyA,
			Aggressor: msg.Side_Sell,
		},
		PassiveOrderId:    passiveOrderId,
		AggressiveOrderId: aggressiveOrderId,
	}

	err = newOrderStore.Post(*passiveOrder)
	assert.Nil(t, err)
	err = newOrderStore.Post(*aggressiveOrder)
	assert.Nil(t, err)
	err = newTradeStore.Post(*trade)
	assert.Nil(t, err)

	positions, err = tradeService.GetPositionsByParty(ctx, testPartyA)
	assert.Nil(t, err)

	fmt.Printf("positions returned:\n")
	for _, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(1000), val.RealisedVolume)
		assert.Equal(t, int64(10000), val.RealisedPNL)
		assert.Equal(t, int64(0), val.UnrealisedVolume)
		assert.Equal(t, int64(0), val.UnrealisedPNL)
	}

	positions, err = tradeService.GetPositionsByParty(ctx, testPartyB)
	assert.Nil(t, err)

	fmt.Printf("positions returned:\n")
	for _, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(1000), val.RealisedVolume)
		assert.Equal(t, int64(-10000), val.RealisedPNL)
		assert.Equal(t, int64(0), val.UnrealisedVolume)
		assert.Equal(t, int64(0), val.UnrealisedPNL)
	}

}