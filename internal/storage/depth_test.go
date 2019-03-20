package storage

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestMarketDepth_Hard(t *testing.T) {
	ctx := context.Background()
	config, err := NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	orderStore, err := NewOrderStore(config)
	assert.Nil(t, err)
	defer orderStore.Close()

	firstBatchOfOrders := []*types.Order{
		{
			Id:        "01",
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     111,
			Remaining: 100,
		},
		{
			Id:        "02",
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     112,
			Remaining: 100,
		},
		{
			Id:        "03",
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     112,
			Remaining: 100,
		},
		{
			Id:        "04",
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     113,
			Remaining: 100,
		},
	}

	for idx := range firstBatchOfOrders {
		orderStore.Post(*firstBatchOfOrders[idx])
	}

	orderStore.Commit()

	marketDepth, _ := orderStore.GetMarketDepth(ctx, testMarket)

	assert.Equal(t, uint64(113), marketDepth.Buy[0].Price)
	assert.Equal(t, uint64(100), marketDepth.Buy[0].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[0].NumberOfOrders)
	assert.Equal(t, uint64(100), marketDepth.Buy[0].CumulativeVolume)

	assert.Equal(t, uint64(112), marketDepth.Buy[1].Price)
	assert.Equal(t, uint64(200), marketDepth.Buy[1].Volume)
	assert.Equal(t, uint64(2), marketDepth.Buy[1].NumberOfOrders)
	assert.Equal(t, uint64(300), marketDepth.Buy[1].CumulativeVolume)

	assert.Equal(t, uint64(111), marketDepth.Buy[2].Price)
	assert.Equal(t, uint64(100), marketDepth.Buy[2].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[2].NumberOfOrders)
	assert.Equal(t, uint64(400), marketDepth.Buy[2].CumulativeVolume)

	secondBatchOfOrders := []*types.Order{
		{
			Id:        "05",
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     111,
			Remaining: 100,
		},
		{
			Id:        "06",
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     112,
			Remaining: 100,
		},
		{
			Id:        "07",
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     112,
			Remaining: 100,
		},
		{
			Id:        "08",
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     113,
			Remaining: 100,
		},
	}

	for idx := range secondBatchOfOrders {
		orderStore.Post(*secondBatchOfOrders[idx])
	}

	// No commit - should remain unchanged

	marketDepth, _ = orderStore.GetMarketDepth(ctx, testMarket)

	assert.Equal(t, uint64(113), marketDepth.Buy[0].Price)
	assert.Equal(t, uint64(100), marketDepth.Buy[0].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[0].NumberOfOrders)
	assert.Equal(t, uint64(100), marketDepth.Buy[0].CumulativeVolume)

	assert.Equal(t, uint64(112), marketDepth.Buy[1].Price)
	assert.Equal(t, uint64(200), marketDepth.Buy[1].Volume)
	assert.Equal(t, uint64(2), marketDepth.Buy[1].NumberOfOrders)
	assert.Equal(t, uint64(300), marketDepth.Buy[1].CumulativeVolume)

	assert.Equal(t, uint64(111), marketDepth.Buy[2].Price)
	assert.Equal(t, uint64(100), marketDepth.Buy[2].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[2].NumberOfOrders)
	assert.Equal(t, uint64(400), marketDepth.Buy[2].CumulativeVolume)

	// COMMIT OK, double the values

	orderStore.Commit()

	marketDepth, _ = orderStore.GetMarketDepth(ctx, testMarket)

	assert.Equal(t, uint64(113), marketDepth.Buy[0].Price)
	assert.Equal(t, uint64(200), marketDepth.Buy[0].Volume)
	assert.Equal(t, uint64(2), marketDepth.Buy[0].NumberOfOrders)
	assert.Equal(t, uint64(200), marketDepth.Buy[0].CumulativeVolume)

	assert.Equal(t, uint64(112), marketDepth.Buy[1].Price)
	assert.Equal(t, uint64(400), marketDepth.Buy[1].Volume)
	assert.Equal(t, uint64(4), marketDepth.Buy[1].NumberOfOrders)
	assert.Equal(t, uint64(600), marketDepth.Buy[1].CumulativeVolume)

	assert.Equal(t, uint64(111), marketDepth.Buy[2].Price)
	assert.Equal(t, uint64(200), marketDepth.Buy[2].Volume)
	assert.Equal(t, uint64(2), marketDepth.Buy[2].NumberOfOrders)
	assert.Equal(t, uint64(800), marketDepth.Buy[2].CumulativeVolume)

	// OK REMOVE

	firstBatchOfOrders[0].Remaining = 0
	firstBatchOfOrders[1].Remaining = firstBatchOfOrders[1].Remaining - 50
	firstBatchOfOrders[2].Remaining = firstBatchOfOrders[2].Remaining - 80
	firstBatchOfOrders[3].Remaining = firstBatchOfOrders[3].Remaining - 100

	for idx := range firstBatchOfOrders {
		orderStore.Put(*firstBatchOfOrders[idx])
	}

	orderStore.Commit()

	marketDepth, _ = orderStore.GetMarketDepth(ctx, testMarket)

	assert.Equal(t, uint64(113), marketDepth.Buy[0].Price)
	assert.Equal(t, uint64(200-100), marketDepth.Buy[0].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[0].NumberOfOrders)
	assert.Equal(t, uint64(200-100), marketDepth.Buy[0].CumulativeVolume)

	assert.Equal(t, uint64(112), marketDepth.Buy[1].Price)
	assert.Equal(t, uint64(400-50-80), marketDepth.Buy[1].Volume)
	assert.Equal(t, uint64(4), marketDepth.Buy[1].NumberOfOrders)
	assert.Equal(t, uint64(600-100-50-80), marketDepth.Buy[1].CumulativeVolume)

	assert.Equal(t, uint64(111), marketDepth.Buy[2].Price)
	assert.Equal(t, uint64(200-100), marketDepth.Buy[2].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[2].NumberOfOrders)
	assert.Equal(t, uint64(800-100-50-80-100), marketDepth.Buy[2].CumulativeVolume)

	// OK REMOVE ALL FROM THE FIRST BATCH
	firstBatchOfOrders[1].Remaining = firstBatchOfOrders[1].Remaining - 50
	firstBatchOfOrders[2].Remaining = firstBatchOfOrders[2].Remaining - 20
	orderStore.Put(*firstBatchOfOrders[1])
	orderStore.Put(*firstBatchOfOrders[2])

	orderStore.Commit()

	marketDepth, _ = orderStore.GetMarketDepth(ctx, testMarket)

	assert.Equal(t, uint64(113), marketDepth.Buy[0].Price)
	assert.Equal(t, uint64(100), marketDepth.Buy[0].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[0].NumberOfOrders)
	assert.Equal(t, uint64(100), marketDepth.Buy[0].CumulativeVolume)

	assert.Equal(t, uint64(112), marketDepth.Buy[1].Price)
	assert.Equal(t, uint64(200), marketDepth.Buy[1].Volume)
	assert.Equal(t, uint64(2), marketDepth.Buy[1].NumberOfOrders)
	assert.Equal(t, uint64(300), marketDepth.Buy[1].CumulativeVolume)

	assert.Equal(t, uint64(111), marketDepth.Buy[2].Price)
	assert.Equal(t, uint64(100), marketDepth.Buy[2].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[2].NumberOfOrders)
	assert.Equal(t, uint64(400), marketDepth.Buy[2].CumulativeVolume)

	// OK REMOVE ALL FROM THE SECOND BATCH TOO MUCH
	secondBatchOfOrders[0].Remaining = secondBatchOfOrders[0].Remaining - uint64(100)

	secondBatchOfOrders[1].Remaining = 0
	secondBatchOfOrders[2].Status = types.Order_Cancelled
	secondBatchOfOrders[3].Status = types.Order_Expired

	for idx := range secondBatchOfOrders {
		orderStore.Put(*secondBatchOfOrders[idx])
	}

	orderStore.Commit()

	marketDepth, _ = orderStore.GetMarketDepth(ctx, testMarket)

	assert.Equal(t, 0, len(marketDepth.Buy))
}

func TestOrderBookDepth_Soft(t *testing.T) {

	var marketDepth marketDepth

	ordersList := []types.Order{
		{Id: "01", Side: types.Side_Buy, Price: 116, Remaining: 100},
		{Id: "02", Side: types.Side_Buy, Price: 110, Remaining: 100},
		{Id: "03", Side: types.Side_Buy, Price: 111, Remaining: 100},
		{Id: "04", Side: types.Side_Buy, Price: 111, Remaining: 100},
		{Id: "05", Side: types.Side_Buy, Price: 113, Remaining: 100},
		{Id: "06", Side: types.Side_Buy, Price: 114, Remaining: 100},
		{Id: "07", Side: types.Side_Buy, Price: 116, Remaining: 100},
	}

	for _, elem := range ordersList {
		marketDepth.Update(elem)
	}

	assert.Equal(t, marketDepth.Buy[0].Price, uint64(116))
	assert.Equal(t, marketDepth.Buy[0].Volume, uint64(200))
	assert.Equal(t, marketDepth.Buy[0].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Buy[0].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[1].Price, uint64(114))
	assert.Equal(t, marketDepth.Buy[1].Volume, uint64(100))
	assert.Equal(t, marketDepth.Buy[1].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Buy[1].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[2].Price, uint64(113))
	assert.Equal(t, marketDepth.Buy[2].Volume, uint64(100))
	assert.Equal(t, marketDepth.Buy[2].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Buy[2].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[3].Price, uint64(111))
	assert.Equal(t, marketDepth.Buy[3].Volume, uint64(200))
	assert.Equal(t, marketDepth.Buy[3].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Buy[3].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[4].Price, uint64(110))
	assert.Equal(t, marketDepth.Buy[4].Volume, uint64(100))
	assert.Equal(t, marketDepth.Buy[4].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Buy[4].CumulativeVolume, uint64(0))

	marketDepth.Update(types.Order{Id: "03", Side: types.Side_Buy, Price: 111, Remaining: 50})
	marketDepth.Update(types.Order{Id: "06", Side: types.Side_Buy, Price: 114, Remaining: 80})
	marketDepth.Update(types.Order{Id: "05", Side: types.Side_Buy, Price: 113, Remaining: 0})

	assert.Equal(t, marketDepth.Buy[0].Price, uint64(116))
	assert.Equal(t, marketDepth.Buy[0].Volume, uint64(200))
	assert.Equal(t, marketDepth.Buy[0].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Buy[0].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[1].Price, uint64(114))
	assert.Equal(t, marketDepth.Buy[1].Volume, uint64(80))
	assert.Equal(t, marketDepth.Buy[1].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Buy[1].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[2].Price, uint64(111))
	assert.Equal(t, marketDepth.Buy[2].Volume, uint64(150))
	assert.Equal(t, marketDepth.Buy[2].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Buy[2].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[3].Price, uint64(110))
	assert.Equal(t, marketDepth.Buy[3].Volume, uint64(100))
	assert.Equal(t, marketDepth.Buy[3].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Buy[3].CumulativeVolume, uint64(0))

	// test sell side
	ordersList = []types.Order{
		{Id: "10", Side: types.Side_Sell, Price: 123, Remaining: 100},
		{Id: "11", Side: types.Side_Sell, Price: 119, Remaining: 100},
		{Id: "12", Side: types.Side_Sell, Price: 120, Remaining: 100},
		{Id: "13", Side: types.Side_Sell, Price: 120, Remaining: 100},
		{Id: "14", Side: types.Side_Sell, Price: 121, Remaining: 100},
		{Id: "15", Side: types.Side_Sell, Price: 121, Remaining: 100},
		{Id: "16", Side: types.Side_Sell, Price: 122, Remaining: 100},
		{Id: "17", Side: types.Side_Sell, Price: 123, Remaining: 100},
	}

	for _, elem := range ordersList {
		marketDepth.Update(elem)
	}

	assert.Equal(t, marketDepth.Sell[0].Price, uint64(119))
	assert.Equal(t, marketDepth.Sell[0].Volume, uint64(100))
	assert.Equal(t, marketDepth.Sell[0].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Sell[0].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[1].Price, uint64(120))
	assert.Equal(t, marketDepth.Sell[1].Volume, uint64(200))
	assert.Equal(t, marketDepth.Sell[1].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Sell[1].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[2].Price, uint64(121))
	assert.Equal(t, marketDepth.Sell[2].Volume, uint64(200))
	assert.Equal(t, marketDepth.Sell[2].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Sell[2].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[3].Price, uint64(122))
	assert.Equal(t, marketDepth.Sell[3].Volume, uint64(100))
	assert.Equal(t, marketDepth.Sell[3].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Sell[3].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[4].Price, uint64(123))
	assert.Equal(t, marketDepth.Sell[4].Volume, uint64(200))
	assert.Equal(t, marketDepth.Sell[4].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Sell[4].CumulativeVolume, uint64(0))

	marketDepth.Update(types.Order{Id: "11", Side: types.Side_Sell, Price: 119, Remaining: 50})
	marketDepth.Update(types.Order{Id: "12", Side: types.Side_Sell, Price: 120, Remaining: 80})
	marketDepth.Update(types.Order{Id: "16", Side: types.Side_Sell, Price: 122, Remaining: 0})

	assert.Equal(t, marketDepth.Sell[0].Price, uint64(119))
	assert.Equal(t, marketDepth.Sell[0].Volume, uint64(50))
	assert.Equal(t, marketDepth.Sell[0].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Sell[0].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[1].Price, uint64(120))
	assert.Equal(t, marketDepth.Sell[1].Volume, uint64(180))
	assert.Equal(t, marketDepth.Sell[1].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Sell[1].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[2].Price, uint64(121))
	assert.Equal(t, marketDepth.Sell[2].Volume, uint64(200))
	assert.Equal(t, marketDepth.Sell[2].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Sell[2].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[3].Price, uint64(123))
	assert.Equal(t, marketDepth.Sell[3].Volume, uint64(200))
	assert.Equal(t, marketDepth.Sell[3].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Sell[3].CumulativeVolume, uint64(0))
}

func TestOrderBookDepthBuySide(t *testing.T) {
	// Scenario:

	// POST few orders to storage
	// call getMarketDepth and see if order book depth is OK

	// create impacted orders and call PUT on them
	// call getMarketDepth and see if order book depth is OK

	// call DELETE on orders
	// call getMarketDepth and see if order book depth is OK

	ctx := context.Background()
	//var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
	config, err := NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	orderStore, err := NewOrderStore(config)
	assert.Nil(t, err)
	defer orderStore.Close()

	orders := []*types.Order{
		{
			Id:        fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     111,
			Remaining: 100,
		},
		{
			Id:        fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     112,
			Remaining: 100,
		},
		{
			Id:        fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     112,
			Remaining: 100,
		},
		{
			Id:        fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     113,
			Remaining: 100,
		},
	}

	for idx := range orders {
		orderStore.Post(*orders[idx])
	}

	orderStore.Commit()

	marketDepth, _ := orderStore.GetMarketDepth(ctx, testMarket)

	assert.Equal(t, uint64(113), marketDepth.Buy[0].Price)
	assert.Equal(t, uint64(100), marketDepth.Buy[0].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[0].NumberOfOrders)
	assert.Equal(t, uint64(100), marketDepth.Buy[0].CumulativeVolume)

	assert.Equal(t, uint64(112), marketDepth.Buy[1].Price)
	assert.Equal(t, uint64(200), marketDepth.Buy[1].Volume)
	assert.Equal(t, uint64(2), marketDepth.Buy[1].NumberOfOrders)
	assert.Equal(t, uint64(300), marketDepth.Buy[1].CumulativeVolume)

	assert.Equal(t, uint64(111), marketDepth.Buy[2].Price)
	assert.Equal(t, uint64(100), marketDepth.Buy[2].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[2].NumberOfOrders)
	assert.Equal(t, uint64(400), marketDepth.Buy[2].CumulativeVolume)

	ordersUpdate := []*types.Order{
		{
			Id:        orders[0].Id,
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     111,
			Remaining: 50,
		},
		{
			Id:        orders[2].Id,
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     112,
			Remaining: 50,
		},
		{
			Id:        orders[3].Id,
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     113,
			Remaining: 80,
			Status:    types.Order_Expired,
		},
	}

	for idx := range ordersUpdate {
		orderStore.Put(*ordersUpdate[idx])
	}

	orderStore.Commit()

	marketDepth, _ = orderStore.GetMarketDepth(ctx, testMarket)

	// 113 is removed

	assert.Equal(t, uint64(112), marketDepth.Buy[0].Price)
	assert.Equal(t, uint64(150), marketDepth.Buy[0].Volume)
	assert.Equal(t, uint64(2), marketDepth.Buy[0].NumberOfOrders)
	assert.Equal(t, uint64(150), marketDepth.Buy[0].CumulativeVolume)

	assert.Equal(t, uint64(111), marketDepth.Buy[1].Price)
	assert.Equal(t, uint64(50), marketDepth.Buy[1].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[1].NumberOfOrders)
	assert.Equal(t, uint64(200), marketDepth.Buy[1].CumulativeVolume)
}

func TestOrderBookDepthSellSide(t *testing.T) {
	// Scenario:

	// POST few orders to storage
	// call getMarketDepth and see if order book depth is OK

	// create impacted orders and call PUT on them
	// call getMarketDepth and see if order book depth is OK

	// call DELETE on orders
	// call getMarketDepth and see if order book depth is OK

	ctx := context.Background()
	//var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
	config, err := NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	orderStore, err := NewOrderStore(config)
	assert.Nil(t, err)
	defer orderStore.Close()

	orders := []*types.Order{
		{
			Id:        fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side:      types.Side_Sell,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     111,
			Remaining: 100,
		},
		{
			Id:        fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side:      types.Side_Sell,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     112,
			Remaining: 100,
		},
		{
			Id:        fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side:      types.Side_Sell,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     112,
			Remaining: 100,
		},
		{
			Id:        fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side:      types.Side_Sell,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     113,
			Remaining: 100,
		},
	}

	for idx := range orders {
		orderStore.Post(*orders[idx])
	}

	orderStore.Commit()

	marketDepth, _ := orderStore.GetMarketDepth(ctx, testMarket)

	assert.Equal(t, uint64(111), marketDepth.Sell[0].Price)
	assert.Equal(t, uint64(100), marketDepth.Sell[0].Volume)
	assert.Equal(t, uint64(1), marketDepth.Sell[0].NumberOfOrders)
	assert.Equal(t, uint64(100), marketDepth.Sell[0].CumulativeVolume)

	assert.Equal(t, uint64(112), marketDepth.Sell[1].Price)
	assert.Equal(t, uint64(200), marketDepth.Sell[1].Volume)
	assert.Equal(t, uint64(2), marketDepth.Sell[1].NumberOfOrders)
	assert.Equal(t, uint64(300), marketDepth.Sell[1].CumulativeVolume)

	assert.Equal(t, uint64(113), marketDepth.Sell[2].Price)
	assert.Equal(t, uint64(100), marketDepth.Sell[2].Volume)
	assert.Equal(t, uint64(1), marketDepth.Sell[2].NumberOfOrders)
	assert.Equal(t, uint64(400), marketDepth.Sell[2].CumulativeVolume)

	ordersUpdate := []*types.Order{
		{
			Id:        orders[0].Id,
			Side:      types.Side_Sell,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     111,
			Remaining: 50,
		},
		{
			Id:        orders[2].Id,
			Side:      types.Side_Sell,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     112,
			Remaining: 50,
		},
		{
			Id:        orders[3].Id,
			Side:      types.Side_Sell,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     113,
			Remaining: 80,
			Status:    types.Order_Expired,
		},
	}

	for idx := range ordersUpdate {
		orderStore.Put(*ordersUpdate[idx])
	}

	orderStore.Commit()

	marketDepth, _ = orderStore.GetMarketDepth(ctx, testMarket)

	assert.Equal(t, uint64(111), marketDepth.Sell[0].Price)
	assert.Equal(t, uint64(50), marketDepth.Sell[0].Volume)
	assert.Equal(t, uint64(1), marketDepth.Sell[0].NumberOfOrders)
	assert.Equal(t, uint64(50), marketDepth.Sell[0].CumulativeVolume)

	assert.Equal(t, uint64(112), marketDepth.Sell[1].Price)
	assert.Equal(t, uint64(150), marketDepth.Sell[1].Volume)
	assert.Equal(t, uint64(2), marketDepth.Sell[1].NumberOfOrders)
	assert.Equal(t, uint64(200), marketDepth.Sell[1].CumulativeVolume)

	// 113 is removed
	assert.Equal(t, 2, len(marketDepth.Sell))

	invalidNewOrders := []*types.Order{
		{
			Id:        "98",
			Side:      types.Side_Buy,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     1337,
			Remaining: 0,
		},
		{
			Id:        "99",
			Side:      types.Side_Sell,
			Market:    testMarket,
			Party:     testPartyA,
			Price:     1337,
			Remaining: 0,
		},
	}

	for idx := range invalidNewOrders {
		orderStore.Post(*invalidNewOrders[idx])
	}
	orderStore.Commit()

	// 1337s did not get added to either side, they're invalid
	assert.Equal(t, 0, len(marketDepth.Buy))
	assert.Equal(t, 2, len(marketDepth.Sell))
}
