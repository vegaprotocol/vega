package storage_test

import (
	"context"
	"strings"
	"testing"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"
	"github.com/stretchr/testify/assert"
)

func TestStorage_NewTrades(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)

	tradeStore, err := storage.NewTrades(logging.NewTestLogger(), config, func() {})
	assert.NotNil(t, tradeStore)
	assert.Nil(t, err)

	config.TradesDirPath = ""

	tradeStore, err = storage.NewTrades(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, tradeStore)
	assert.NotNil(t, err)

	nsf := strings.Contains(err.Error(), "no such file or directory")
	assert.True(t, nsf)
}

func TestStorage_GetTradesByOrderId(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)

	orderStore, err := storage.NewOrders(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, err)

	tradeStore, err := storage.NewTrades(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, err)

	defer orderStore.Close()
	defer tradeStore.Close()

	insertTestData(t, orderStore, tradeStore)
	trades, err := tradeStore.GetByOrderID(context.Background(), "d41d8cd98f00b204e9800998ecf9999a", 0, 0, false, nil)

	assert.Nil(t, err)
	assert.Equal(t, 6, len(trades))
	assert.Equal(t, "trade-id-1", trades[0].Id)
	assert.Equal(t, "trade-id-2", trades[1].Id)
	assert.Equal(t, "trade-id-3", trades[2].Id)
	assert.Equal(t, "trade-id-4", trades[3].Id)
	assert.Equal(t, "trade-id-5", trades[4].Id)
	assert.Equal(t, "trade-id-6", trades[5].Id)
}

func TestStorage_GetTradesByPartyWithPagination(t *testing.T) {
	ctx := context.Background()
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)

	orderStore, err := storage.NewOrders(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, err)

	tradeStore, err := storage.NewTrades(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, err)

	defer orderStore.Close()
	defer tradeStore.Close()

	insertTestData(t, orderStore, tradeStore)

	// Want last 3 trades (timestamp descending)
	last := uint64(3)

	// Expect 3 trades with descending trade-ids
	trades, err := tradeStore.GetByParty(ctx, testPartyA, 0, last, true, nil)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, "trade-id-6", trades[0].Id)
	assert.Equal(t, "trade-id-5", trades[1].Id)
	assert.Equal(t, "trade-id-4", trades[2].Id)

	// Want last 3 trades (timestamp descending) and skip 2
	skip := uint64(2)

	// Expect 3 trades with descending trade-ids
	trades, err = tradeStore.GetByParty(ctx, testPartyA, skip, last, true, nil)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, "trade-id-4", trades[0].Id)
	assert.Equal(t, "trade-id-3", trades[1].Id)
	assert.Equal(t, "trade-id-2", trades[2].Id)
}

func TestStorage_GetTradesByMarketWithPagination(t *testing.T) {
	ctx := context.Background()
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)

	orderStore, err := storage.NewOrders(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, err)

	tradeStore, err := storage.NewTrades(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, err)

	defer orderStore.Close()
	defer tradeStore.Close()

	insertTestData(t, orderStore, tradeStore)

	// Expect 6 trades with no filtration/pagination
	trades, err := tradeStore.GetByMarket(ctx, testMarket, 0, 0, false)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(trades))

	// Want first 2 trades (timestamp ascending)
	first := uint64(2)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, 0, first, false)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(trades))
	assert.Equal(t, "trade-id-1", trades[0].Id)
	assert.Equal(t, "trade-id-2", trades[1].Id)

	// Want last 3 trades (timestamp descending)
	last := uint64(3)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, 0, last, true)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, "trade-id-6", trades[0].Id)
	assert.Equal(t, "trade-id-5", trades[1].Id)
	assert.Equal(t, "trade-id-4", trades[2].Id)

	// Want first 2 trades after skipping 2
	skip := uint64(2)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, skip, first, false)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(trades))
	assert.Equal(t, "trade-id-3", trades[0].Id)
	assert.Equal(t, "trade-id-4", trades[1].Id)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, skip, last, true)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, "trade-id-4", trades[0].Id)
	assert.Equal(t, "trade-id-3", trades[1].Id)
	assert.Equal(t, "trade-id-2", trades[2].Id)

	// Skip a large page size of trades (compared to our set)
	// effectively skipping past the end of the set, so no
	// trades should be available at that offset
	skip = uint64(50)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, skip, last, true)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(trades))

}

func insertTestData(t *testing.T, orderStore *storage.Order, tradeStore *storage.Trade) {

	// Arrange seed orders & trades
	orderA := &types.Order{
		Id:          "d41d8cd98f00b204e9800998ecf9999a",
		MarketID:    testMarket,
		PartyID:     testPartyA,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        1000,
		Remaining:   1000,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		CreatedAt:   0,
		Status:      types.Order_STATUS_ACTIVE,
	}

	orderB := &types.Order{
		Id:          "d41d8cd98f00b204e9800998ecf8427h",
		MarketID:    testMarket,
		PartyID:     testPartyB,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		CreatedAt:   1,
		Status:      types.Order_STATUS_ACTIVE,
	}

	trade1 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        "trade-id-1",
		Price:     100,
		Size:      100,
		MarketID:  testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: types.Side_SIDE_SELL,
		Timestamp: 1,
		BuyOrder:  orderB.Id,
		SellOrder: orderA.Id,
	}

	trade2 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        "trade-id-2",
		Price:     100,
		Size:      100,
		MarketID:  testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: types.Side_SIDE_SELL,
		Timestamp: 1,
		BuyOrder:  orderB.Id,
		SellOrder: orderA.Id,
	}

	trade3 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        "trade-id-3",
		Price:     100,
		Size:      100,
		MarketID:  testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: types.Side_SIDE_SELL,
		Timestamp: 1,
		BuyOrder:  orderB.Id,
		SellOrder: orderA.Id,
	}

	trade4 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        "trade-id-4",
		Price:     100,
		Size:      100,
		MarketID:  testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: types.Side_SIDE_SELL,
		Timestamp: 1,
		BuyOrder:  orderB.Id,
		SellOrder: orderA.Id,
	}

	trade5 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        "trade-id-5",
		Price:     100,
		Size:      100,
		MarketID:  testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: types.Side_SIDE_SELL,
		Timestamp: 1,
		BuyOrder:  orderB.Id,
		SellOrder: orderA.Id,
	}

	trade6 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        "trade-id-6",
		Price:     100,
		Size:      100,
		MarketID:  testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: types.Side_SIDE_SELL,
		Timestamp: 1,
		BuyOrder:  orderB.Id,
		SellOrder: orderA.Id,
	}

	// Add orders
	err := orderStore.SaveBatch([]types.Order{*orderA, *orderB})
	assert.NoError(t, err)

	// Add trades
	err = tradeStore.SaveBatch([]types.Trade{*trade1, *trade2, *trade3, *trade4, *trade5, *trade6})
	assert.NoError(t, err)
}
