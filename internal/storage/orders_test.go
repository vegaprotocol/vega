package storage_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/config/encoding"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

const testMarket = "market"
const testParty = "party"
const testPartyA = "partyA"
const testPartyB = "partyB"

func TestStorage_NewOrders(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)

	orderStore, err := storage.NewOrders(logging.NewTestLogger(), config, func() {})
	assert.NotNil(t, orderStore)
	assert.Nil(t, err)

	config.OrderStoreDirPath = ""

	orderStore, err = storage.NewOrders(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, orderStore)
	assert.NotNil(t, err)

	nsf := strings.Contains(err.Error(), "no such file or directory")
	assert.True(t, nsf)
}

func TestStorage_PostAndGetNewOrder(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)
	orderStore, err := storage.NewOrders(logging.NewTestLogger(), config, func() {})
	defer orderStore.Close()

	var order = &types.Order{
		Id:       "45305210ff7a9bb9450b1833cc10368a",
		MarketID: "testMarket",
		PartyID:  "testParty",
	}

	err = orderStore.Post(*order)
	assert.Nil(t, err)

	orderStore.Commit()

	o, err := orderStore.GetByMarketAndId(context.Background(), "testMarket", order.Id)
	assert.Nil(t, err)
	assert.Equal(t, order.Id, o.Id)
}

func TestStorage_GetOrdersForMarket(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)

	var tests = []struct {
		inMarkets      []string
		inOrders       []*types.Order
		inLimit        uint64
		inMarket       string
		outOrdersCount int
	}{
		{
			inMarkets: []string{"testMarket1", "marketZ"},
			inOrders: []*types.Order{
				{
					Id:       "d41d8cd98f00b204e9800998ecf8427e",
					MarketID: "testMarket1",
					PartyID:  testParty,
				},
				{
					Id:       "ad2dc275947362c45893bbeb30fc3098",
					MarketID: "marketZ",
					PartyID:  testParty,
				},
				{
					Id:       "4e8e41367997cfe705d62ea80592cbcc",
					MarketID: "testMarket1",
					PartyID:  testParty,
				},
			},
			inLimit:        5000,
			inMarket:       "testMarket1",
			outOrdersCount: 2,
		},
		{
			inMarkets: []string{testMarket, "marketABC"},
			inOrders: []*types.Order{
				{
					Id:       "d41d8cd98f00b204e9800998ecf8427e",
					MarketID: testMarket,
					PartyID:  testParty,
				},
				{
					Id:       "ad2dc275947362c45893bbeb30fc3098",
					MarketID: "marketABC",
					PartyID:  testParty,
				},
				{
					Id:       "4e8e41367997cfe705d62ea80592cbcc",
					MarketID: testMarket,
					PartyID:  testParty,
				},
			},
			inLimit:        5000,
			inMarket:       "marketABC",
			outOrdersCount: 1,
		},
		{
			inMarkets: []string{"marketXYZ"},
			inOrders: []*types.Order{
				{
					Id:       "d41d8cd98f00b204e9800998ecf8427e",
					MarketID: "marketXYZ",
					PartyID:  testParty,
				},
				{
					Id:       "ad2dc275947362c45893bbeb30fc3098",
					MarketID: "marketXYZ",
					PartyID:  testParty,
				},
				{
					Id:       "4e8e41367997cfe705d62ea80592cbcc",
					MarketID: "marketXYZ",
					PartyID:  testParty,
				},
			},
			inLimit:        2,
			inMarket:       "marketXYZ",
			outOrdersCount: 2,
		},
	}

	for _, tt := range tests {
		orderStore, err := storage.NewOrders(logging.NewTestLogger(), config, func() {})
		assert.Nil(t, err)

		for _, order := range tt.inOrders {
			err := orderStore.Post(*order)
			assert.Nil(t, err)
		}

		orderStore.Commit()

		orders, err := orderStore.GetByMarket(context.Background(), tt.inMarket, 0, tt.inLimit, false, nil)
		assert.Nil(t, err)
		assert.Equal(t, tt.outOrdersCount, len(orders))
		orderStore.Close()
	}
}

func TestStorage_GetOrdersForParty(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)

	orderStore, err := storage.NewOrders(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, err)
	defer orderStore.Close()

	passiveOrder := &types.Order{
		Id:        "d41d8cd98f00b204e9800998ecf9999e",
		MarketID:  testMarket,
		PartyID:   testPartyA,
		Remaining: 0,
	}

	aggressiveOrder := &types.Order{
		Id:        "d41d8cd98f00b204e9800998ecf8427e",
		MarketID:  testMarket,
		PartyID:   testPartyB,
		Remaining: 100,
	}

	err = orderStore.Post(*passiveOrder)
	assert.Nil(t, err)

	err = orderStore.Post(*aggressiveOrder)
	assert.Nil(t, err)

	orderStore.Commit()

	ordersAtPartyA, err := orderStore.GetByParty(context.Background(), testPartyA, 0, 0, false, nil)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(ordersAtPartyA))

	ordersAtPartyB, err := orderStore.GetByParty(context.Background(), testPartyB, 0, 0, false, nil)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(ordersAtPartyB))

	orderAtPartyA, err := orderStore.GetByPartyAndId(context.Background(), testPartyA, passiveOrder.Id)
	assert.Nil(t, err)
	assert.Equal(t, passiveOrder.Id, orderAtPartyA.Id)

	orderAtPartyB, err := orderStore.GetByPartyAndId(context.Background(), testPartyB, aggressiveOrder.Id)
	assert.Nil(t, err)
	assert.Equal(t, aggressiveOrder.Id, orderAtPartyB.Id)

	// update order, parties should also be updated as its a pointer
	updatedAggressiveOrder := &types.Order{
		Id:        "d41d8cd98f00b204e9800998ecf8427e",
		MarketID:  testMarket,
		PartyID:   testPartyB,
		Remaining: 0,
	}

	err = orderStore.Put(*updatedAggressiveOrder)
	assert.Nil(t, err)
	orderAtPartyB, err = orderStore.GetByPartyAndId(context.Background(), testPartyB, aggressiveOrder.Id)
	assert.Nil(t, err)
	assert.Equal(t, updatedAggressiveOrder.Id, orderAtPartyB.Id)
}

func TestStorage_GetOrderByReference(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	log := logging.NewTestLogger()

	storage.FlushStores(log, config)
	newOrderStore, err := storage.NewOrders(log, config, func() {})
	assert.Nil(t, err)
	defer newOrderStore.Close()

	order := &types.Order{
		Id:        "d41d8cd98f00b204e9800998ecf8427b",
		MarketID:  testMarket,
		PartyID:   testPartyA,
		Side:      types.Side_Buy,
		Price:     100,
		Size:      1000,
		Remaining: 0,
		Type:      types.Order_GTC,
		CreatedAt: 0,
		Status:    types.Order_Active,
		Reference: "123123-34334343-1231231",
	}

	err = newOrderStore.Post(*order)
	assert.Nil(t, err)

	newOrderStore.Commit()

	fetchedOrder, err := newOrderStore.GetByParty(context.Background(), testPartyA, 0, 1, true, nil)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(fetchedOrder))
	assert.Equal(t, order.Id, fetchedOrder[0].Id)
}

// Ensures that we return a market depth struct with empty buy/sell for
// markets that have no orders (when they are newly created)
func TestStorage_GetMarketDepthForNewMarket(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}
	log := logging.NewTestLogger()
	storage.FlushStores(log, config)
	orderStore, err := storage.NewOrders(log, config, func() {})
	assert.Nil(t, err)
	defer orderStore.Close()

	depth, err := orderStore.GetMarketDepth(context.Background(), testMarket)
	assert.Nil(t, err)

	assert.Equal(t, testMarket, depth.MarketID)
	assert.Equal(t, 0, len(depth.Buy))
	assert.Equal(t, 0, len(depth.Sell))
}

// Ensure market depth returns expected price levels from incoming orders
func TestStorage_GetMarketDepth(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}
	log := logging.NewTestLogger()
	storage.FlushStores(log, config)
	orderStore, err := storage.NewOrders(log, config, func() {})
	assert.Nil(t, err)
	defer orderStore.Close()

	order1 := &types.Order{
		Id:        "d41d8cd98f00b204e9800998ecf8427b",
		MarketID:  testMarket,
		PartyID:   testPartyA,
		Side:      types.Side_Buy,
		Price:     100,
		Size:      1000,
		Remaining: 1000,
		Type:      types.Order_GTC,
		CreatedAt: 0,
		Status:    types.Order_Active,
		Reference: "123123-34334343-1231231",
	}

	order2 := &types.Order{
		Id:        "d41d8cd98f00b204e9800998ecf8427c",
		MarketID:  testMarket,
		PartyID:   testPartyA,
		Side:      types.Side_Buy,
		Price:     100,
		Size:      1000,
		Remaining: 1000,
		Type:      types.Order_GTC,
		CreatedAt: 0,
		Status:    types.Order_Active,
		Reference: "123123-34334343-1231232",
	}

	order3 := &types.Order{
		Id:        "d41d8cd98f00b204e9800998hhf8427c",
		MarketID:  testMarket,
		PartyID:   testPartyB,
		Side:      types.Side_Sell,
		Price:     9999,
		Size:      20,
		Remaining: 20,
		Type:      types.Order_GTC,
		CreatedAt: 0,
		Status:    types.Order_Active,
		Reference: "123123-34334343-1231232",
	}

	err = orderStore.Post(*order1)
	assert.Nil(t, err)

	err = orderStore.Post(*order2)
	assert.Nil(t, err)

	err = orderStore.Post(*order3)
	assert.Nil(t, err)

	err = orderStore.Commit()
	assert.Nil(t, err)

	depth, err := orderStore.GetMarketDepth(context.Background(), testMarket)
	assert.Nil(t, err)

	assert.Equal(t, testMarket, depth.MarketID)
	assert.Equal(t, 1, len(depth.Buy))
	assert.Equal(t, 1, len(depth.Sell))
	assert.Equal(t, uint64(100), depth.Buy[0].Price)
	assert.Equal(t, uint64(9999), depth.Sell[0].Price)
}

func TestStorage_GetMarketDepthWithTimeout(t *testing.T) {
	ctx := context.Background()
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}
	config.Timeout = encoding.Duration{Duration: time.Nanosecond}
	log := logging.NewTestLogger()
	storage.FlushStores(log, config)
	orderStore, err := storage.NewOrders(log, config, func() {})
	assert.Nil(t, err)
	defer orderStore.Close()

	order := &types.Order{
		Id:        "d41d8cd98f00b204e9800998ecf8427b",
		MarketID:  testMarket,
		PartyID:   testPartyA,
		Side:      types.Side_Buy,
		Price:     100,
		Size:      1000,
		Remaining: 1000,
		Type:      types.Order_GTC,
		CreatedAt: 0,
		Status:    types.Order_Active,
		Reference: "123123-34334343-1231231",
	}

	err = orderStore.Post(*order)
	assert.Nil(t, err)

	err = orderStore.Commit()
	assert.Nil(t, err)

	// Bit of a hacky test, but we want to test timeouts when getting market depth because we can only set a timeout
	// of 1s or more through config, we're setting a timeout of 1 nanosecond on the context we pass to orderStore
	// this ensures that the context will get cancelled when getting market depth, and that code path gets tested
	tctx, cfunc := context.WithTimeout(ctx, time.Nanosecond)
	defer cfunc()

	// perhaps sleep here in case we need to make sure the context has indeed expired, but starting the 2 routines and the map lookups
	// alone will take longer than a nanosecond anyway, so there's no need.
	_, err = orderStore.GetMarketDepth(tctx, testMarket)
	assert.Equal(t, storage.ErrTimeoutReached, err)
}
