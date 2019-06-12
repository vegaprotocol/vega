package storage_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	storcfg "code.vegaprotocol.io/vega/internal/storage/config"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testMarket = "market"
const testParty = "party"
const testPartyA = "partyA"
const testPartyB = "partyB"

func NewTestOrdersConfig(t *testing.T) storcfg.OrdersConfig {
	return storcfg.NewDefaultOrdersConfig(tempDir(t, "testorderstore"))
}

func noop() {}

func runOrderStoreTest(t *testing.T, test func(t *testing.T, orderStore *storage.Order)) {
	log := logging.NewTestLogger()
	cfg := NewTestOrdersConfig(t)
	s, err := storage.NewOrders(log, cfg, noop)
	assert.NotNil(t, s)
	require.NoError(t, err)
	defer os.RemoveAll(cfg.Storage.Path)
	defer s.Close()

	test(t, s)
}

func TestStorage_NewOrders(t *testing.T) {
	runOrderStoreTest(t, func(t *testing.T, orderStore *storage.Order) {})
}

func TestStorage_NewOrders_BadDir(t *testing.T) {
	log := logging.NewTestLogger()
	cfg := NewTestOrdersConfig(t)
	cfg.Storage.Path = ""
	s, err := storage.NewOrders(log, cfg, noop)
	assert.Nil(t, s)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "no such file or directory"))
}

func TestStorage_PostAndGetNewOrder(t *testing.T) {
	runOrderStoreTest(t, func(t *testing.T, orderStore *storage.Order) {
		var order = &types.Order{
			Id:       "45305210ff7a9bb9450b1833cc10368a",
			MarketID: "testMarket",
			PartyID:  "testParty",
		}

		err := orderStore.Post(*order)
		assert.Nil(t, err)

		orderStore.Commit()

		o, err := orderStore.GetByMarketAndId(context.Background(), "testMarket", order.Id)
		assert.Nil(t, err)
		assert.Equal(t, order.Id, o.Id)
	})
}

func TestStorage_PostAndGetByReference(t *testing.T) {
	runOrderStoreTest(t, func(t *testing.T, orderStore *storage.Order) {
		var order = &types.Order{
			Reference: "83cfdf76-8eac-4c7e-8f6a-2aa51e89364f",
			Id:        "45305210ff7a9bb9450b1833cc10368a",
			MarketID:  "testMarket",
			PartyID:   "testParty",
		}

		err := orderStore.Post(*order)
		assert.Nil(t, err)

		orderStore.Commit()

		o, err := orderStore.GetByReference(context.Background(), order.Reference)
		assert.Nil(t, err)
		assert.Equal(t, order.Id, o.Id)
	})
}

func TestStorage_GetOrdersForMarket(t *testing.T) {
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
		runOrderStoreTest(t, func(t *testing.T, orderStore *storage.Order) {
			for _, order := range tt.inOrders {
				err := orderStore.Post(*order)
				assert.Nil(t, err)
			}

			orderStore.Commit()

			orders, err := orderStore.GetByMarket(context.Background(), tt.inMarket, 0, tt.inLimit, false, nil)
			assert.Nil(t, err)
			assert.Equal(t, tt.outOrdersCount, len(orders))
		})
	}
}

func TestStorage_GetOrdersForParty(t *testing.T) {
	runOrderStoreTest(t, func(t *testing.T, orderStore *storage.Order) {
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

		err := orderStore.Post(*passiveOrder)
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
	})
}

func TestStorage_GetOrderByReference(t *testing.T) {
	runOrderStoreTest(t, func(t *testing.T, orderStore *storage.Order) {
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

		err := orderStore.Post(*order)
		assert.Nil(t, err)

		orderStore.Commit()

		fetchedOrder, err := orderStore.GetByParty(context.Background(), testPartyA, 0, 1, true, nil)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(fetchedOrder))
		assert.Equal(t, order.Id, fetchedOrder[0].Id)
	})
}

// Ensures that we return a market depth struct with empty buy/sell for
// markets that have no orders (when they are newly created)
func TestStorage_GetMarketDepthForNewMarket(t *testing.T) {
	runOrderStoreTest(t, func(t *testing.T, orderStore *storage.Order) {
		depth, err := orderStore.GetMarketDepth(context.Background(), testMarket)
		assert.Nil(t, err)

		assert.Equal(t, testMarket, depth.MarketID)
		assert.Equal(t, 0, len(depth.Buy))
		assert.Equal(t, 0, len(depth.Sell))
	})
}

// Ensure market depth returns expected price levels from incoming orders
func TestStorage_GetMarketDepth(t *testing.T) {
	runOrderStoreTest(t, func(t *testing.T, orderStore *storage.Order) {
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

		err := orderStore.Post(*order1)
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
	})
}

func TestStorage_GetMarketDepthWithTimeout(t *testing.T) {
	runOrderStoreTest(t, func(t *testing.T, orderStore *storage.Order) {
		ctx := context.Background()
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

		err := orderStore.Post(*order)
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
	})
}
