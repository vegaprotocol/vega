package datastore

import (
	"testing"
	"vega/msg"
	"github.com/stretchr/testify/assert"
	"vega/filters"
)

func TestMemTradeStore_GetByPartyWithPagination(t *testing.T) {
	 newTradeStore, _ := buildPaginationTestTrades(t)

	// Expect 6 trades with no filtration/pagination
	trades, err := newTradeStore.GetByParty(testPartyA, nil)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(trades))
}

func TestMemTradeStore_GetByMarketWithPagination(t *testing.T) {
	newTradeStore, _ := buildPaginationTestTrades(t)

	// Expect 6 trades with no filtration/pagination
	trades, err := newTradeStore.GetByMarket(testMarket, nil)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(trades))

	// Want first 2 trades (timestamp ascending)
	first := uint64(2)
	queryFilters := &filters.TradeQueryFilters{}
	queryFilters.First = &first

	trades, err = newTradeStore.GetByMarket(testMarket, queryFilters)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(trades))
	assert.Equal(t, "trade-id-1", trades[0].Id)
	assert.Equal(t, "trade-id-2", trades[1].Id)

	// Want last 3 trades (timestamp descending)
	last := uint64(3)
	queryFilters = &filters.TradeQueryFilters{}
	queryFilters.Last = &last

	trades, err = newTradeStore.GetByMarket(testMarket, queryFilters)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, "trade-id-6", trades[0].Id)
	assert.Equal(t, "trade-id-5", trades[1].Id)
	assert.Equal(t, "trade-id-4", trades[2].Id)

	// Want first 2 trades after skipping 2
	skip := uint64(2)
	queryFilters = &filters.TradeQueryFilters{}
	queryFilters.First = &first
	queryFilters.Skip = &skip

	trades, err = newTradeStore.GetByMarket(testMarket, queryFilters)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(trades))
	assert.Equal(t, "trade-id-3", trades[0].Id)
	assert.Equal(t, "trade-id-4", trades[1].Id)

	// Want last 3 trades after skipping 2
	queryFilters = &filters.TradeQueryFilters{}
	queryFilters.Last = &last
	queryFilters.Skip = &skip

	trades, err = newTradeStore.GetByMarket(testMarket, queryFilters)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, "trade-id-4", trades[0].Id)
	assert.Equal(t, "trade-id-3", trades[1].Id)
	assert.Equal(t, "trade-id-2", trades[2].Id)

	// Skip a large page size of trades (compared to our set)
	// effectively skipping past the end of the set, so no
	// trades should be available at that offset
	skip = uint64(50)
	queryFilters = &filters.TradeQueryFilters{}
	queryFilters.Last = &last
	queryFilters.Skip = &skip

	trades, err = newTradeStore.GetByMarket(testMarket, queryFilters)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(trades))

}

func buildPaginationTestTrades(t *testing.T) (TradeStore, OrderStore) {
	var memStore = NewMemStore([]string{testMarket}, []string{})
	var newOrderStore = NewOrderStore(&memStore)
	defer newOrderStore.Close()
	var newTradeStore = NewTradeStore(&memStore)

	// Arrange seed orders & trades
	orderA := Order{
		Order: msg.Order{
			Id:         "d41d8cd98f00b204e9800998ecf9999a",
			Market:     testMarket,
			Party:      testPartyA,
			Side:       msg.Side_Sell,
			Price:      100,
			Size:       1000,
			Remaining:  1000,
			Type:       msg.Order_GTC,
			Timestamp:  0,
			Status:     msg.Order_Active,
		},
	}

	orderB := Order{
		Order: msg.Order{
			Id:         "d41d8cd98f00b204e9800998ecf8427h",
			Market:     testMarket,
			Party:      testPartyB,
			Side:       msg.Side_Buy,
			Price:      100,
			Size:       100,
			Remaining:  100,
			Type:       msg.Order_GTC,
			Timestamp:  1,
			Status:     msg.Order_Active,
		},
	}

	trade1 := Trade{
		Trade: msg.Trade{
			Id:        "trade-id-1",
			Price:     100,
			Size:      100,
			Market:    testMarket,
			Buyer:     testPartyB,
			Seller:    testPartyA,
			Aggressor: msg.Side_Sell,
			Timestamp: 1,
		},
		AggressiveOrderId: orderA.Order.Id,
		PassiveOrderId:    orderB.Order.Id,
	}

	trade2 := Trade{
		Trade: msg.Trade{
			Id:        "trade-id-2",
			Price:     100,
			Size:      100,
			Market:    testMarket,
			Buyer:     testPartyB,
			Seller:    testPartyA,
			Aggressor: msg.Side_Sell,
			Timestamp: 1,
		},
		AggressiveOrderId: orderA.Order.Id,
		PassiveOrderId:    orderB.Order.Id,
	}

	trade3 := Trade{
		Trade: msg.Trade{
			Id:        "trade-id-3",
			Price:     100,
			Size:      100,
			Market:    testMarket,
			Buyer:     testPartyB,
			Seller:    testPartyA,
			Aggressor: msg.Side_Sell,
			Timestamp: 1,
		},
		AggressiveOrderId: orderA.Order.Id,
		PassiveOrderId:    orderB.Order.Id,
	}

	trade4 := Trade{
		Trade: msg.Trade{
			Id:        "trade-id-4",
			Price:     100,
			Size:      100,
			Market:    testMarket,
			Buyer:     testPartyB,
			Seller:    testPartyA,
			Aggressor: msg.Side_Sell,
			Timestamp: 1,
		},
		AggressiveOrderId: orderA.Order.Id,
		PassiveOrderId:    orderB.Order.Id,
	}

	trade5 := Trade{
		Trade: msg.Trade{
			Id:        "trade-id-5",
			Price:     100,
			Size:      100,
			Market:    testMarket,
			Buyer:     testPartyB,
			Seller:    testPartyA,
			Aggressor: msg.Side_Sell,
			Timestamp: 1,
		},
		AggressiveOrderId: orderA.Order.Id,
		PassiveOrderId:    orderB.Order.Id,
	}

	trade6 := Trade{
		Trade: msg.Trade{
			Id:        "trade-id-6",
			Price:     100,
			Size:      100,
			Market:    testMarket,
			Buyer:     testPartyB,
			Seller:    testPartyA,
			Aggressor: msg.Side_Sell,
			Timestamp: 1,
		},
		AggressiveOrderId: orderA.Order.Id,
		PassiveOrderId:    orderB.Order.Id,
	}

	// Add orders
	err := newOrderStore.Post(orderA)
	assert.Nil(t, err)
	err = newOrderStore.Post(orderB)
	assert.Nil(t, err)

	// Add trades
	err = newTradeStore.Post(trade1)
	assert.Nil(t, err)
	err = newTradeStore.Post(trade2)
	assert.Nil(t, err)
	err = newTradeStore.Post(trade3)
	assert.Nil(t, err)
	err = newTradeStore.Post(trade4)
	assert.Nil(t, err)
	err = newTradeStore.Post(trade5)
	assert.Nil(t, err)
	err = newTradeStore.Post(trade6)
	assert.Nil(t, err)

	return newTradeStore, newOrderStore
}
