package storage

import (
	"testing"
	"vega/msg"
	"github.com/stretchr/testify/assert"
	"vega/filters"
	"os"
	"vega/internal/logging"
	"strings"
)

const testMarket = "market"
const testParty  = "party"
const testPartyA = "partyA"
const testPartyB = "partyB"

func defaultConfig() *Config {
	// Test logger can be configured here, default to console not file etc.
	logger := logging.NewLogger()
	logger.InitConsoleLogger(logging.DebugLevel)
	logger.AddExitHandler()
	// Test configuration for badger stores
	return &Config{
		orderStoreDirPath: "./tmp/test-orderstore",
		tradeStoreDirPath: "./tmp/test-tradestore",
		log: logger,
	}
}

func flushStores(c *Config) {
	err := os.RemoveAll(c.orderStoreDirPath)
	if err != nil {
		c.log.Errorf("error flushing order store: %s", err)
	}
	err = os.RemoveAll(c.tradeStoreDirPath)
	if err != nil {
		c.log.Errorf("error flushing trade store: %s", err)
	}
}

func TestStorage_NewOrderStore(t *testing.T) {
	config := defaultConfig()
	flushStores(config)

	orderStore, err := NewOrderStore(config)
	assert.NotNil(t, orderStore)
	assert.Nil(t, err)

	config.orderStoreDirPath = ""

	orderStore, err = NewOrderStore(config)
	assert.Nil(t, orderStore)
	assert.NotNil(t, err)

	nsf := strings.Contains(err.Error(), "no such file or directory")
	assert.True(t, nsf)
}

func TestStorage_PostAndGetNewOrder(t *testing.T) {
	config := defaultConfig()
	flushStores(config)
	orderStore, err := NewOrderStore(config)
	defer orderStore.Close()

	var order = &msg.Order{
		Id:     "45305210ff7a9bb9450b1833cc10368a",
		Market: "testMarket",
		Party:  "testParty",
	}

	err = orderStore.Post(*order)
	assert.Nil(t, err)

	orderStore.Commit()

	o, err := orderStore.GetByMarketAndId("testMarket", order.Id)
	assert.Nil(t, err)
	assert.Equal(t, order.Id, o.Id)
}

func TestStorage_GetOrdersForMarket(t *testing.T) {
	config := defaultConfig()
	flushStores(config)

	var tests = []struct {
		inMarkets      []string
		inOrders       []*msg.Order
		inLimit        uint64
		inMarket       string
		outOrdersCount int
	}{
		{
			inMarkets: []string{"testMarket1", "marketZ"},
			inOrders: []*msg.Order{
				{
						Id:     "d41d8cd98f00b204e9800998ecf8427e",
						Market: "testMarket1",
						Party:  testParty,
				},
				{
						Id:     "ad2dc275947362c45893bbeb30fc3098",
						Market: "marketZ",
						Party:  testParty,
				},
				{
						Id:     "4e8e41367997cfe705d62ea80592cbcc",
						Market: "testMarket1",
						Party:  testParty,
				},
			},
			inLimit:        5000,
			inMarket:       "testMarket1",
			outOrdersCount: 2,
		},
		{
			inMarkets: []string{testMarket, "marketABC"},
			inOrders: []*msg.Order{
				{
						Id:     "d41d8cd98f00b204e9800998ecf8427e",
						Market: testMarket,
						Party:  testParty,
				},
				{
						Id:     "ad2dc275947362c45893bbeb30fc3098",
						Market: "marketABC",
						Party:  testParty,
				},
				{
						Id:     "4e8e41367997cfe705d62ea80592cbcc",
						Market: testMarket,
						Party:  testParty,
				},
			},
			inLimit:        5000,
			inMarket:       "marketABC",
			outOrdersCount: 1,
		},
		{
			inMarkets: []string{"marketXYZ"},
			inOrders: []*msg.Order{
				{
						Id:     "d41d8cd98f00b204e9800998ecf8427e",
						Market: "marketXYZ",
						Party:  testParty,
				},
				{
						Id:     "ad2dc275947362c45893bbeb30fc3098",
						Market: "marketXYZ",
						Party:  testParty,
				},
				{
						Id:     "4e8e41367997cfe705d62ea80592cbcc",
						Market: "marketXYZ",
						Party:  testParty,
				},
			},
			inLimit:        2,
			inMarket:       "marketXYZ",
			outOrdersCount: 2,
		},
	}

	for _, tt := range tests {
		orderStore, err := NewOrderStore(config)
		assert.Nil(t, err)

		for _, order := range tt.inOrders {
			err := orderStore.Post(*order)
			assert.Nil(t, err)
		}

		orderStore.Commit()

		f := &filters.OrderQueryFilters{}
		f.First = &tt.inLimit
		orders, err := orderStore.GetByMarket(tt.inMarket, f)
		assert.Nil(t, err)
		assert.Equal(t, tt.outOrdersCount, len(orders))
		orderStore.Close()
	}
}

func TestStorage_GetOrdersForParty(t *testing.T) {
	config := defaultConfig()
	flushStores(config)

	var orderStore, err = NewOrderStore(config)
	assert.Nil(t, err)
	defer orderStore.Close()

	passiveOrder := &msg.Order{
			Id:        "d41d8cd98f00b204e9800998ecf9999e",
			Market:    testMarket,
			Party:     testPartyA,
			Remaining: 0,
	}

	aggressiveOrder := &msg.Order{
			Id:        "d41d8cd98f00b204e9800998ecf8427e",
			Market:    testMarket,
			Party:     testPartyB,
			Remaining: 100,
	}

	err = orderStore.Post(*passiveOrder)
	assert.Nil(t, err)

	err = orderStore.Post(*aggressiveOrder)
	assert.Nil(t, err)

	orderStore.Commit()

	ordersAtPartyA, err := orderStore.GetByParty(testPartyA, nil)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(ordersAtPartyA))

	ordersAtPartyB, err := orderStore.GetByParty(testPartyB, nil)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(ordersAtPartyB))

	orderAtPartyA, err := orderStore.GetByPartyAndId(testPartyA, passiveOrder.Id)
	assert.Nil(t, err)
	assert.Equal(t, passiveOrder.Id, orderAtPartyA.Id)

	orderAtPartyB, err := orderStore.GetByPartyAndId(testPartyB, aggressiveOrder.Id)
	assert.Nil(t, err)
	assert.Equal(t, aggressiveOrder.Id, orderAtPartyB.Id)

	// update order, parties should also be updated as its a pointer
	updatedAggressiveOrder := &msg.Order{
			Id:        "d41d8cd98f00b204e9800998ecf8427e",
			Market:    testMarket,
			Party:     testPartyB,
			Remaining: 0,
	}

	err = orderStore.Put(*updatedAggressiveOrder)
	assert.Nil(t, err)
	orderAtPartyB, err = orderStore.GetByPartyAndId(testPartyB, aggressiveOrder.Id)
	assert.Nil(t, err)
	assert.Equal(t, updatedAggressiveOrder.Id, orderAtPartyB.Id)
}

func TestStorage_OrderFiltration(t *testing.T) {
	config := defaultConfig()
	flushStores(config)

	orderStore, err := NewOrderStore(config)
	assert.Nil(t, err)
	defer orderStore.Close()

	order1 := &msg.Order{
			Id:         "d41d8cd98f00b204e9800998ecf9999a",
			Market:     testMarket,
			Party:      testPartyA,
			Side:       msg.Side_Sell,
			Price:      100,
			Size:       1000,
			Remaining:  0,
			Type:       msg.Order_GTC,
			Timestamp:  0,
			Status:     msg.Order_Active,
	}

	order2 := &msg.Order{
			Id:         "d41d8cd98f00b204e9800998ecf8427b",
			Market:     testMarket,
			Party:      testPartyB,
			Side:       msg.Side_Buy,
			Price:      110,
			Size:       900,
			Remaining:  0,
			Type:       msg.Order_GTC,
			Timestamp:  0,
			Status:     msg.Order_Active,
	}

	order3 := &msg.Order{
			Id:         "d41d8cd98f00b204e9800998ecf8427c",
			Market:     testMarket,
			Party:      testPartyA,
			Side:       msg.Side_Buy,
			Price:      1000,
			Size:       1000,
			Remaining:  1000,
			Type:       msg.Order_GTC,
			Timestamp:  1,
			Status:     msg.Order_Cancelled,
	}

	order4 := &msg.Order{
			Id:         "d41d8cd98f00b204e9800998ecf8427d",
			Market:     testMarket,
			Party:      testPartyA,
			Side:       msg.Side_Sell,
			Price:      100,
			Size:       100,
			Remaining:  100,
			Type:       msg.Order_GTC,
			Timestamp:  1,
			Status:     msg.Order_Active,
	}

	// check if db empty
	orders, err := orderStore.GetByMarket(testMarket, nil)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(orders))

	// add orders
	err = orderStore.Post(*order1)
	assert.Nil(t, err)
	err = orderStore.Post(*order2)
	assert.Nil(t, err)
	err = orderStore.Post(*order3)
	assert.Nil(t, err)
	err = orderStore.Post(*order4)
	assert.Nil(t, err)

	orderStore.Commit()

	orderFilters := &filters.OrderQueryFilters{
		MarketFilter: &filters.QueryFilter{Eq: testMarket},
		PartyFilter:  &filters.QueryFilter{Eq: testPartyA},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(orders))

	// get all orders
	orderFilters = &filters.OrderQueryFilters{
		MarketFilter: &filters.QueryFilter{Eq: testMarket},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		PriceFilter: &filters.QueryFilter{FilterRange: &filters.QueryFilterRange{Lower: uint64(150), Upper: uint64(1150)}, Kind: "uint64"},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		PriceFilter: &filters.QueryFilter{FilterRange: &filters.QueryFilterRange{Lower: uint64(99), Upper: uint64(200)}, Kind: "uint64"},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		RemainingFilter: &filters.QueryFilter{FilterRange: &filters.QueryFilterRange{Lower: uint64(1), Upper: uint64(10000)}, Kind: "uint64"},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		RemainingFilter: &filters.QueryFilter{FilterRange: &filters.QueryFilterRange{Lower: uint64(0), Upper: uint64(10000)}, Kind: "uint64"},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		SizeFilter: &filters.QueryFilter{Eq: uint64(900)},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		SizeFilter: &filters.QueryFilter{Neq: uint64(900)},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		TypeFilter: &filters.QueryFilter{Eq: msg.Order_GTC},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		TypeFilter: &filters.QueryFilter{Neq: msg.Order_GTC},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		PriceFilter: &filters.QueryFilter{Eq: uint64(1000)},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		TypeFilter:  &filters.QueryFilter{Neq: msg.Order_GTC},
		PriceFilter: &filters.QueryFilter{Eq: uint64(1000)},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		TimestampFilter: &filters.QueryFilter{Eq: uint64(1)},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		TimestampFilter: &filters.QueryFilter{FilterRange: &filters.QueryFilterRange{Lower: uint64(1), Upper: uint64(10)}, Kind: "uint64"},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		TimestampFilter: &filters.QueryFilter{FilterRange: &filters.QueryFilterRange{Lower: uint64(5), Upper: uint64(10)}, Kind: "uint64"},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		TimestampFilter: &filters.QueryFilter{FilterRange: &filters.QueryFilterRange{Lower: uint64(0), Upper: uint64(10)}, Kind: "uint64"},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		StatusFilter: &filters.QueryFilter{Eq: msg.Order_Active},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		StatusFilter: &filters.QueryFilter{Eq: msg.Order_Cancelled},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		StatusFilter: &filters.QueryFilter{Eq: msg.Order_Expired},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		StatusFilter: &filters.QueryFilter{Neq: msg.Order_Expired},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(orders))

	orderFilters = &filters.OrderQueryFilters{
		IdFilter: &filters.QueryFilter{ Eq: "d41d8cd98f00b204e9800998ecf8427c"},
	}
	orders, err = orderStore.GetByMarket(testMarket, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(orders))
}

func TestStorage_GetOrderByReference(t *testing.T) {
	config := defaultConfig()
	flushStores(config)
	newOrderStore, err := NewOrderStore(config)
	assert.Nil(t, err)
	defer newOrderStore.Close()

	order := &msg.Order{
			Id:         "d41d8cd98f00b204e9800998ecf8427b",
			Market:     testMarket,
			Party:      testPartyA,
			Side:       msg.Side_Buy,
			Price:      100,
			Size:       1000,
			Remaining:  0,
			Type:       msg.Order_GTC,
			Timestamp:  0,
			Status:     msg.Order_Active,
			Reference:  "123123-34334343-1231231",
	}

	err = newOrderStore.Post(*order)
	assert.Nil(t, err)

	newOrderStore.Commit()

	orderFilters := &filters.OrderQueryFilters{
		ReferenceFilter: &filters.QueryFilter{ Eq: "123123-34334343-1231231"},
	}

	fetchedOrder, err := newOrderStore.GetByParty(testPartyA, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(fetchedOrder))
	assert.Equal(t, order.Id, fetchedOrder[0].Id)
}

func TestStorage_InsertBatchOrders(t *testing.T) {
	config := defaultConfig()
	flushStores(config)
	orderStore, err := NewOrderStore(config)
	assert.Nil(t, err)
	defer orderStore.Close()

	order1 := &msg.Order{
		Id:         "d41d8cd98f00b204e9800998ecf8427b",
		Market:     testMarket,
		Party:      testPartyA,
		Side:       msg.Side_Buy,
		Price:      100,
		Size:       1000,
		Remaining:  0,
		Type:       msg.Order_GTC,
		Timestamp:  0,
		Status:     msg.Order_Active,
		Reference:  "123123-34334343-1231231",
	}

	order2 := &msg.Order{
		Id:         "d41d8cd98f00b204e9800998ecf8427c",
		Market:     testMarket,
		Party:      testPartyA,
		Side:       msg.Side_Buy,
		Price:      100,
		Size:       1000,
		Remaining:  0,
		Type:       msg.Order_GTC,
		Timestamp:  0,
		Status:     msg.Order_Active,
		Reference:  "123123-34334343-1231232",
	}

	err = orderStore.Post(*order1)
	assert.Nil(t, err)

	err = orderStore.Post(*order2)
	assert.Nil(t, err)

	orderFilters := &filters.OrderQueryFilters{
		ReferenceFilter: &filters.QueryFilter{ Eq: "123123-34334343-1231231"},
	}

	fetchedOrder, err := orderStore.GetByParty(testPartyA, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(fetchedOrder))

	orderStore.Commit()

	fetchedOrder, err = orderStore.GetByParty(testPartyA, orderFilters)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(fetchedOrder))
	assert.Equal(t, order1.Id, fetchedOrder[0].Id)
}
