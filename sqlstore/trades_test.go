package sqlstore_test

import (
	"context"
	"testing"
	"time"

	types "code.vegaprotocol.io/protos/vega"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/stretchr/testify/assert"
)

const (
	testMarket = "b4376d805a888548baabfae74ef6f4fa4680dc9718bab355fa7191715de4fafe"
	testPartyA = "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"
	testPartyB = "521127F24B1FA40311BA2FB3F6977310346346604B275DB7B767B04240A5A5C3"
	orderAId   = "787B72CB5DD7A5EA869E49F361CF957DF747F849B4ACE88ABC6DA0F9C450AFDD"
	orderBId   = "83dc82be23c77daec384a239143f07f83c667acf60d734745b023c6567e7b57b"

	tradeId1 = "0bd678723c33b059638953e0904d2ddbd78c2be72ab25a8753a622911c2d9c78"
	tradeId2 = "af2bb48edd738353fcd7a2b6cea4821dd2382ec95497954535278dfbfff7b5b5"
	tradeId3 = "3d4ed10064b7cedbc8a37316f7329f853c9588b6a55006ffb8bec3f1a4ccc88e"
	tradeId4 = "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"
	tradeId5 = "8b6be1a03cc4d529f682887a78b66e6879d17f81e2b37356ca0acbc5d5886eb8"
	tradeId6 = "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"
)

func TestStorage_GetTradesByOrderId(t *testing.T) {
	market := testMarket

	GetTradesByOrderIdAndMarket(t, &market)
	GetTradesByOrderIdAndMarket(t, nil)
}

func GetTradesByOrderIdAndMarket(t *testing.T, market *string) {
	defer testStore.DeleteEverything()

	tradeStore := sqlstore.NewTrades(testStore)

	insertTestData(t, tradeStore)

	trades, err := tradeStore.GetByOrderID(context.Background(), orderAId, market, entities.Pagination{})

	assert.Nil(t, err)
	assert.Equal(t, 6, len(trades))
	assert.Equal(t, tradeId1, trades[0].ID.String())
	assert.Equal(t, tradeId2, trades[1].ID.String())
	assert.Equal(t, tradeId3, trades[2].ID.String())
	assert.Equal(t, tradeId4, trades[3].ID.String())
	assert.Equal(t, tradeId5, trades[4].ID.String())
	assert.Equal(t, tradeId6, trades[5].ID.String())
}

func TestStorage_GetTradesByPartyWithPagination(t *testing.T) {
	market := testMarket
	GetTradesByPartyAndMarketWithPagination(t, &market)
	GetTradesByPartyAndMarketWithPagination(t, nil)
}

func GetTradesByPartyAndMarketWithPagination(t *testing.T, market *string) {
	ctx := context.Background()
	defer testStore.DeleteEverything()

	tradeStore := sqlstore.NewTrades(testStore)

	insertTestData(t, tradeStore)

	// Want last 3 trades (timestamp descending)
	last := uint64(3)

	// Expect 3 trades with descending trade-ids

	trades, err := tradeStore.GetByParty(ctx, testPartyA, market, entities.Pagination{Limit: last, Descending: true})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, tradeId6, trades[0].ID.String())
	assert.Equal(t, tradeId5, trades[1].ID.String())
	assert.Equal(t, tradeId4, trades[2].ID.String())

	// Want last 3 trades (timestamp descending) and skip 2
	skip := uint64(2)

	// Expect 3 trades with descending trade-ids
	trades, err = tradeStore.GetByParty(ctx, testPartyA, market, entities.Pagination{Skip: skip, Limit: last, Descending: true})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, tradeId4, trades[0].ID.String())
	assert.Equal(t, tradeId3, trades[1].ID.String())
	assert.Equal(t, tradeId2, trades[2].ID.String())
}

func TestStorage_GetTradesByMarketWithPagination(t *testing.T) {
	ctx := context.Background()

	defer testStore.DeleteEverything()

	tradeStore := sqlstore.NewTrades(testStore)

	insertTestData(t, tradeStore)

	// Expect 6 trades with no filtration/pagination
	trades, err := tradeStore.GetByMarket(ctx, testMarket, entities.Pagination{})
	assert.Nil(t, err)
	assert.Equal(t, 6, len(trades))

	// Want first 2 trades (timestamp ascending)
	first := uint64(2)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, entities.Pagination{Limit: first})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(trades))
	assert.Equal(t, tradeId1, trades[0].ID.String())
	assert.Equal(t, tradeId2, trades[1].ID.String())

	// Want last 3 trades (timestamp descending)
	last := uint64(3)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, entities.Pagination{Limit: last, Descending: true})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, tradeId6, trades[0].ID.String())
	assert.Equal(t, tradeId5, trades[1].ID.String())
	assert.Equal(t, tradeId4, trades[2].ID.String())

	// Want first 2 trades after skipping 2
	skip := uint64(2)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, entities.Pagination{Skip: skip, Limit: first})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(trades))
	assert.Equal(t, tradeId3, trades[0].ID.String())
	assert.Equal(t, tradeId4, trades[1].ID.String())

	trades, err = tradeStore.GetByMarket(ctx, testMarket, entities.Pagination{Skip: skip, Limit: last, Descending: true})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, tradeId4, trades[0].ID.String())
	assert.Equal(t, tradeId3, trades[1].ID.String())
	assert.Equal(t, tradeId2, trades[2].ID.String())

	// Skip a large page size of trades (compared to our set)
	// effectively skipping past the end of the set, so no
	// trades should be available at that offset
	skip = uint64(50)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, entities.Pagination{Skip: skip, Limit: last, Descending: true})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(trades))
}

func insertTestData(t *testing.T, tradeStore *sqlstore.Trades) {
	bs := sqlstore.NewBlocks(testStore)
	now := time.Now()
	block1 := addTestBlockForTime(t, bs, now)
	block2 := addTestBlockForTime(t, bs, now.Add(time.Second))

	trade1 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        tradeId1,
		Price:     "100",
		Size:      100,
		MarketId:  testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: types.Side_SIDE_SELL,
		Timestamp: 1,
		BuyOrder:  orderBId,
		SellOrder: orderAId,
	}

	trade2 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        tradeId2,
		Price:     "100",
		Size:      100,
		MarketId:  testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: types.Side_SIDE_SELL,
		Timestamp: 1,
		BuyOrder:  orderBId,
		SellOrder: orderAId,
	}

	trade3 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        tradeId3,
		Price:     "100",
		Size:      100,
		MarketId:  testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: types.Side_SIDE_SELL,
		Timestamp: 1,
		BuyOrder:  orderBId,
		SellOrder: orderAId,
	}

	trade4 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        tradeId4,
		Price:     "100",
		Size:      100,
		MarketId:  testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: types.Side_SIDE_SELL,
		Timestamp: 1,
		BuyOrder:  orderBId,
		SellOrder: orderAId,
	}

	trade5 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        tradeId5,
		Price:     "100",
		Size:      100,
		MarketId:  testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: types.Side_SIDE_SELL,
		Timestamp: 1,
		BuyOrder:  orderBId,
		SellOrder: orderAId,
	}

	trade6 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        tradeId6,
		Price:     "100",
		Size:      100,
		MarketId:  testMarket,
		Buyer:     testPartyB,
		Seller:    testPartyA,
		Aggressor: types.Side_SIDE_SELL,
		Timestamp: 1,
		BuyOrder:  orderBId,
		SellOrder: orderAId,
	}

	protos := []types.Trade{*trade1, *trade2, *trade3, *trade4, *trade5, *trade6}

	var seqNum uint64
	vegaTime := block1.VegaTime
	for _, proto := range protos {
		if seqNum == 3 {
			seqNum = 0
			vegaTime = block2.VegaTime
		}
		trade, err := entities.TradeFromProto(&proto, vegaTime, seqNum)
		if err != nil {
			t.Fatalf("failed to get trade from proto:%s", err)
		}
		err = tradeStore.Add(trade)
		if err != nil {
			t.Fatalf("failed to add trade:%s", err)
		}
		seqNum++
	}
}
