package datastore

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"vega/filters"
	"vega/msg"
)

const tradeStoreDir = "../tmp/tradestore"

func FlushTradeStore() {
	err := os.RemoveAll(tradeStoreDir)
	if err != nil {
		fmt.Printf("UNABLE TO FLUSH DB: %s\n", err.Error())
	}
}

func TestMemTradeStore_GetByPartyWithPagination(t *testing.T) {
	FlushTradeStore()
	FlushOrderStore()
	var newOrderStore = NewOrderStore(orderStoreDir)
	defer newOrderStore.Close()
	var newTradeStore = NewTradeStore(tradeStoreDir)
	defer newTradeStore.Close()

	populateStores(t, newOrderStore, newTradeStore)

	// Want last 3 trades (timestamp descending)
	last := uint64(3)
	queryFilters := &filters.TradeQueryFilters{}
	queryFilters.Last = &last

	// Expect 3 trades with descending trade-ids
	trades, err := newTradeStore.GetByParty(testPartyA, queryFilters)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, "trade-id-6", trades[0].Id)
	assert.Equal(t, "trade-id-5", trades[1].Id)
	assert.Equal(t, "trade-id-4", trades[2].Id)

	// Want last 3 trades (timestamp descending) and skip 2
	last = uint64(3)
	skip := uint64(2)
	queryFilters = &filters.TradeQueryFilters{}
	queryFilters.Last = &last
	queryFilters.Skip = &skip

	// Expect 3 trades with descending trade-ids
	trades, err = newTradeStore.GetByParty(testPartyA, queryFilters)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, "trade-id-4", trades[0].Id)
	assert.Equal(t, "trade-id-3", trades[1].Id)
	assert.Equal(t, "trade-id-2", trades[2].Id)
}

func TestMemTradeStore_GetByMarketWithPagination(t *testing.T) {
	FlushTradeStore()
	FlushOrderStore()
	var newOrderStore = NewOrderStore(orderStoreDir)
	defer newOrderStore.Close()
	var newTradeStore = NewTradeStore(tradeStoreDir)
	defer newTradeStore.Close()

	populateStores(t, newOrderStore, newTradeStore)

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

	//// Want last 3 trades (timestamp descending)
	//last := uint64(3)
	//queryFilters = &filters.TradeQueryFilters{}
	//queryFilters.Last = &last
	//
	//trades, err = newTradeStore.GetByMarket(testMarket, queryFilters)
	//assert.Nil(t, err)
	//assert.Equal(t, 3, len(trades))
	//assert.Equal(t, "trade-id-6", trades[0].Id)
	//assert.Equal(t, "trade-id-5", trades[1].Id)
	//assert.Equal(t, "trade-id-4", trades[2].Id)
	//
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
	//queryFilters = &filters.TradeQueryFilters{}
	//queryFilters.Last = &last
	//queryFilters.Skip = &skip
	//
	//trades, err = newTradeStore.GetByMarket(testMarket, queryFilters)
	//assert.Nil(t, err)
	//assert.Equal(t, 3, len(trades))
	//assert.Equal(t, "trade-id-4", trades[0].Id)
	//assert.Equal(t, "trade-id-3", trades[1].Id)
	//assert.Equal(t, "trade-id-2", trades[2].Id)

	// Skip a large page size of trades (compared to our set)
	// effectively skipping past the end of the set, so no
	// trades should be available at that offset
	//skip = uint64(50)
	//queryFilters = &filters.TradeQueryFilters{}
	//queryFilters.Last = &last
	//queryFilters.Skip = &skip
	//
	//trades, err = newTradeStore.GetByMarket(testMarket, queryFilters)
	//assert.Nil(t, err)
	//assert.Equal(t, 0, len(trades))

}

func populateStores(t *testing.T, orderStore OrderStore, tradeStore TradeStore) {

	// Arrange seed orders & trades
	orderA := &msg.Order{
		Id:        "d41d8cd98f00b204e9800998ecf9999a",
		Market:    testMarket,
		Party:     testPartyA,
		Side:      msg.Side_Sell,
		Price:     100,
		Size:      1000,
		Remaining: 1000,
		Type:      msg.Order_GTC,
		Timestamp: 0,
		Status:    msg.Order_Active,
	}

	orderB := &msg.Order{
		Id:        "d41d8cd98f00b204e9800998ecf8427h",
		Market:    testMarket,
		Party:     testPartyB,
		Side:      msg.Side_Buy,
		Price:     100,
		Size:      100,
		Remaining: 100,
		Type:      msg.Order_GTC,
		Timestamp: 1,
		Status:    msg.Order_Active,
	}

	trade1 := &msg.Trade{
		Id:        "trade-id-1",
		Price:     100,
		Size:      100,
		Market:    testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: msg.Side_Sell,
		Timestamp: 1,
		BuyOrder:  orderB.Id,
		SellOrder: orderA.Id,
	}

	trade2 := &msg.Trade{
		Id:        "trade-id-2",
		Price:     100,
		Size:      100,
		Market:    testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: msg.Side_Sell,
		Timestamp: 1,
		BuyOrder:  orderB.Id,
		SellOrder: orderA.Id,
	}

	trade3 := &msg.Trade{
		Id:        "trade-id-3",
		Price:     100,
		Size:      100,
		Market:    testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: msg.Side_Sell,
		Timestamp: 1,
		BuyOrder:  orderB.Id,
		SellOrder: orderA.Id,
	}

	trade4 := &msg.Trade{
		Id:        "trade-id-4",
		Price:     100,
		Size:      100,
		Market:    testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: msg.Side_Sell,
		Timestamp: 1,
		BuyOrder:  orderB.Id,
		SellOrder: orderA.Id,
	}

	trade5 := &msg.Trade{
		Id:        "trade-id-5",
		Price:     100,
		Size:      100,
		Market:    testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: msg.Side_Sell,
		Timestamp: 1,
		BuyOrder:  orderB.Id,
		SellOrder: orderA.Id,
	}

	trade6 := &msg.Trade{
		Id:        "trade-id-6",
		Price:     100,
		Size:      100,
		Market:    testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: msg.Side_Sell,
		Timestamp: 1,
		BuyOrder:  orderB.Id,
		SellOrder: orderA.Id,
	}

	// Add orders
	err := orderStore.Post(orderA)
	assert.Nil(t, err)
	err = orderStore.Post(orderB)
	assert.Nil(t, err)

	// Add trades
	err = tradeStore.Post(trade1)
	assert.Nil(t, err)
	err = tradeStore.Post(trade2)
	assert.Nil(t, err)
	err = tradeStore.Post(trade3)
	assert.Nil(t, err)
	err = tradeStore.Post(trade4)
	assert.Nil(t, err)
	err = tradeStore.Post(trade5)
	assert.Nil(t, err)
	err = tradeStore.Post(trade6)
	assert.Nil(t, err)
}
