// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore_test

import (
	"context"
	"testing"
	"time"

	types "code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/sqlstore"
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
	defer DeleteEverything()

	tradeStore := sqlstore.NewTrades(connectionSource)

	insertTestData(t, tradeStore)

	trades, err := tradeStore.GetByOrderID(context.Background(), orderAId, market, entities.OffsetPagination{})

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
	defer DeleteEverything()

	tradeStore := sqlstore.NewTrades(connectionSource)

	insertTestData(t, tradeStore)

	// Want last 3 trades (timestamp descending)
	last := uint64(3)

	// Expect 3 trades with descending trade-ids

	trades, err := tradeStore.GetByParty(ctx, testPartyA, market, entities.OffsetPagination{Limit: last, Descending: true})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, tradeId6, trades[0].ID.String())
	assert.Equal(t, tradeId5, trades[1].ID.String())
	assert.Equal(t, tradeId4, trades[2].ID.String())

	// Want last 3 trades (timestamp descending) and skip 2
	skip := uint64(2)

	// Expect 3 trades with descending trade-ids
	trades, err = tradeStore.GetByParty(ctx, testPartyA, market, entities.OffsetPagination{Skip: skip, Limit: last, Descending: true})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, tradeId4, trades[0].ID.String())
	assert.Equal(t, tradeId3, trades[1].ID.String())
	assert.Equal(t, tradeId2, trades[2].ID.String())
}

func TestStorage_GetTradesByMarketWithPagination(t *testing.T) {
	ctx := context.Background()

	defer DeleteEverything()

	tradeStore := sqlstore.NewTrades(connectionSource)

	insertTestData(t, tradeStore)

	// Expect 6 trades with no filtration/pagination
	trades, err := tradeStore.GetByMarket(ctx, testMarket, entities.OffsetPagination{})
	assert.Nil(t, err)
	assert.Equal(t, 6, len(trades))

	// Want first 2 trades (timestamp ascending)
	first := uint64(2)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, entities.OffsetPagination{Limit: first})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(trades))
	assert.Equal(t, tradeId1, trades[0].ID.String())
	assert.Equal(t, tradeId2, trades[1].ID.String())

	// Want last 3 trades (timestamp descending)
	last := uint64(3)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, entities.OffsetPagination{Limit: last, Descending: true})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, tradeId6, trades[0].ID.String())
	assert.Equal(t, tradeId5, trades[1].ID.String())
	assert.Equal(t, tradeId4, trades[2].ID.String())

	// Want first 2 trades after skipping 2
	skip := uint64(2)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, entities.OffsetPagination{Skip: skip, Limit: first})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(trades))
	assert.Equal(t, tradeId3, trades[0].ID.String())
	assert.Equal(t, tradeId4, trades[1].ID.String())

	trades, err = tradeStore.GetByMarket(ctx, testMarket, entities.OffsetPagination{Skip: skip, Limit: last, Descending: true})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, tradeId4, trades[0].ID.String())
	assert.Equal(t, tradeId3, trades[1].ID.String())
	assert.Equal(t, tradeId2, trades[2].ID.String())

	// Skip a large page size of trades (compared to our set)
	// effectively skipping past the end of the set, so no
	// trades should be available at that offset
	skip = uint64(50)

	trades, err = tradeStore.GetByMarket(ctx, testMarket, entities.OffsetPagination{Skip: skip, Limit: last, Descending: true})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(trades))
}

func insertTestData(t *testing.T, tradeStore *sqlstore.Trades) {
	bs := sqlstore.NewBlocks(connectionSource)
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

	tradeStore.Flush(context.Background())
}

func TestTrades_CursorPagination(t *testing.T) {
	t.Run("Should return all trades for a given market when no cursor is given", testTradesCursorPaginationByMarketNoCursor)
	t.Run("Should return all trades for a given party when no market and no cursor is given", testTradesCursorPaginationByPartyNoMarketNoCursor)
	t.Run("Should return all trades for a given party and market when market ID and no cursor is given", testTradesCursorPaginationByPartyAndMarketNoCursor)
	t.Run("Should return the first page of trades for a given market when a first cursor is set", testTradesCursorPaginationByMarketWithCursorFirst)
	t.Run("Should return the first page of trades for a given party when a first cursor is set but not market", testTradesCursorPaginationByPartyWithCursorNoMarketFirst)
	t.Run("Should return the first page of trades for a given party and market when a first cursor is set", testTradesCursorPaginationByPartyAndMarketWithCursorFirst)
	t.Run("Should return the last page of trades for a given market when a last cursor is set", testTradesCursorPaginationByMarketWithCursorLast)
	t.Run("Should return the last page of trades for a given party when a last cursor is set but not market", testTradesCursorPaginationByPartyWithCursorNoMarketLast)
	t.Run("Should return the last page of trades for a given party and market when a last cursor is set", testTradesCursorPaginationByPartyAndMarketWithCursorLast)
	t.Run("Should return the page of trades for a given market when a first and after cursor is set", testTradesCursorPaginationByMarketWithCursorForward)
	t.Run("Should return the page of trades for a given party when a first and after cursor is set but not market", testTradesCursorPaginationByPartyWithCursorNoMarketForward)
	t.Run("Should return the page of trades for a given party and market when a first and after cursor is set", testTradesCursorPaginationByPartyAndMarketWithCursorForward)
	t.Run("Should return the page of trades for a given market when a last and before cursor is set", testTradesCursorPaginationByMarketWithCursorBackward)
	t.Run("Should return the page of trades for a given party when a last and before cursor is set but not market", testTradesCursorPaginationByPartyWithCursorNoMarketBackward)
	t.Run("Should return the page of trades for a given party and market when a last and before cursor is set", testTradesCursorPaginationByPartyAndMarketWithCursorBackward)

	t.Run("Should return all trades for a given market when no cursor is given - newest first", testTradesCursorPaginationByMarketNoCursorNewestFirst)
	t.Run("Should return all trades for a given party when no market and no cursor is given - newest first", testTradesCursorPaginationByPartyNoMarketNoCursorNewestFirst)
	t.Run("Should return all trades for a given party and market when market ID and no cursor is given - newest first", testTradesCursorPaginationByPartyAndMarketNoCursorNewestFirst)
	t.Run("Should return the first page of trades for a given market when a first cursor is set - newest first", testTradesCursorPaginationByMarketWithCursorFirstNewestFirst)
	t.Run("Should return the first page of trades for a given party when a first cursor is set but not market - newest first", testTradesCursorPaginationByPartyWithCursorNoMarketFirstNewestFirst)
	t.Run("Should return the first page of trades for a given party and market when a first cursor is set - newest first", testTradesCursorPaginationByPartyAndMarketWithCursorFirstNewestFirst)
	t.Run("Should return the last page of trades for a given market when a last cursor is set - newest first", testTradesCursorPaginationByMarketWithCursorLastNewestFirst)
	t.Run("Should return the last page of trades for a given party when a last cursor is set but not market - newest first", testTradesCursorPaginationByPartyWithCursorNoMarketLastNewestFirst)
	t.Run("Should return the last page of trades for a given party and market when a last cursor is set - newest first", testTradesCursorPaginationByPartyAndMarketWithCursorLastNewestFirst)
	t.Run("Should return the page of trades for a given market when a first and after cursor is set - newest first", testTradesCursorPaginationByMarketWithCursorForwardNewestFirst)
	t.Run("Should return the page of trades for a given party when a first and after cursor is set but not market - newest first", testTradesCursorPaginationByPartyWithCursorNoMarketForwardNewestFirst)
	t.Run("Should return the page of trades for a given party and market when a first and after cursor is set - newest first", testTradesCursorPaginationByPartyAndMarketWithCursorForwardNewestFirst)
	t.Run("Should return the page of trades for a given market when a last and before cursor is set - newest first", testTradesCursorPaginationByMarketWithCursorBackwardNewestFirst)
	t.Run("Should return the page of trades for a given party when a last and before cursor is set but not market - newest first", testTradesCursorPaginationByPartyWithCursorNoMarketBackwardNewestFirst)
	t.Run("Should return the page of trades for a given party and market when a last and before cursor is set - newest first", testTradesCursorPaginationByPartyAndMarketWithCursorBackwardNewestFirst)
}

func setupTradesTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.Trades, sqlstore.Config, func(t *testing.T)) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ts := sqlstore.NewTrades(connectionSource)

	DeleteEverything()

	config := sqlstore.NewDefaultConfig()
	config.ConnectionConfig.Port = testDBPort

	return bs, ts, config, func(t *testing.T) {
		t.Helper()
		DeleteEverything()
	}
}

func populateTestTrades(ctx context.Context, t *testing.T, bs *sqlstore.Blocks, ts *sqlstore.Trades, blockTimes map[string]time.Time) {
	t.Helper()

	trades := []entities.Trade{
		{
			SeqNum:    1,
			ID:        entities.NewTradeID("02a16077"),
			MarketID:  entities.NewMarketID("deadbeef"),
			Price:     decimal.NewFromFloat(1.0),
			Size:      1,
			Buyer:     entities.NewPartyID("dabbad00"),
			Seller:    entities.NewPartyID("facefeed"),
			BuyOrder:  entities.NewOrderID("02a16077"),
			SellOrder: entities.NewOrderID("fb1528a5"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    2,
			ID:        entities.NewTradeID("44eea1bc"),
			MarketID:  entities.NewMarketID("deadbeef"),
			Price:     decimal.NewFromFloat(2.0),
			Size:      2,
			Buyer:     entities.NewPartyID("dabbad00"),
			Seller:    entities.NewPartyID("facefeed"),
			BuyOrder:  entities.NewOrderID("44eea1bc"),
			SellOrder: entities.NewOrderID("da8d1803"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    3,
			ID:        entities.NewTradeID("65be62cd"),
			MarketID:  entities.NewMarketID("deadbeef"),
			Price:     decimal.NewFromFloat(3.0),
			Size:      3,
			Buyer:     entities.NewPartyID("dabbad00"),
			Seller:    entities.NewPartyID("facefeed"),
			BuyOrder:  entities.NewOrderID("65be62cd"),
			SellOrder: entities.NewOrderID("c8744329"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    4,
			ID:        entities.NewTradeID("7a797e0e"),
			MarketID:  entities.NewMarketID("deadbeef"),
			Price:     decimal.NewFromFloat(4.0),
			Size:      4,
			Buyer:     entities.NewPartyID("dabbad00"),
			Seller:    entities.NewPartyID("facefeed"),
			BuyOrder:  entities.NewOrderID("7a797e0e"),
			SellOrder: entities.NewOrderID("c612300d"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    5,
			ID:        entities.NewTradeID("7bb2356e"),
			MarketID:  entities.NewMarketID("cafed00d"),
			Price:     decimal.NewFromFloat(5.0),
			Size:      5,
			Buyer:     entities.NewPartyID("dabbad00"),
			Seller:    entities.NewPartyID("facefeed"),
			BuyOrder:  entities.NewOrderID("7bb2356e"),
			SellOrder: entities.NewOrderID("b7c84b8e"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    6,
			ID:        entities.NewTradeID("b7c84b8e"),
			MarketID:  entities.NewMarketID("cafed00d"),
			Price:     decimal.NewFromFloat(6.0),
			Size:      6,
			Buyer:     entities.NewPartyID("d0d0caca"),
			Seller:    entities.NewPartyID("decafbad"),
			BuyOrder:  entities.NewOrderID("b7c84b8e"),
			SellOrder: entities.NewOrderID("7bb2356e"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    7,
			ID:        entities.NewTradeID("c612300d"),
			MarketID:  entities.NewMarketID("cafed00d"),
			Price:     decimal.NewFromFloat(7.0),
			Size:      7,
			Buyer:     entities.NewPartyID("d0d0caca"),
			Seller:    entities.NewPartyID("decafbad"),
			BuyOrder:  entities.NewOrderID("c612300d"),
			SellOrder: entities.NewOrderID("7a797e0e"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    8,
			ID:        entities.NewTradeID("c8744329"),
			MarketID:  entities.NewMarketID("cafed00d"),
			Price:     decimal.NewFromFloat(8.0),
			Size:      8,
			Buyer:     entities.NewPartyID("d0d0caca"),
			Seller:    entities.NewPartyID("decafbad"),
			BuyOrder:  entities.NewOrderID("c8744329"),
			SellOrder: entities.NewOrderID("65be62cd"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    9,
			ID:        entities.NewTradeID("da8d1803"),
			MarketID:  entities.NewMarketID("deadbaad"),
			Price:     decimal.NewFromFloat(9.0),
			Size:      9,
			Buyer:     entities.NewPartyID("baadf00d"),
			Seller:    entities.NewPartyID("0d15ea5e"),
			BuyOrder:  entities.NewOrderID("da8d1803"),
			SellOrder: entities.NewOrderID("44eea1bc"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    10,
			ID:        entities.NewTradeID("fb1528a5"),
			MarketID:  entities.NewMarketID("deadbaad"),
			Price:     decimal.NewFromFloat(10.0),
			Size:      10,
			Buyer:     entities.NewPartyID("baadf00d"),
			Seller:    entities.NewPartyID("0d15ea5e"),
			BuyOrder:  entities.NewOrderID("fb1528a5"),
			SellOrder: entities.NewOrderID("02a16077"),
			Type:      entities.TradeTypeDefault,
		},
	}

	for _, td := range trades {
		trade := td
		block := addTestBlock(t, bs)
		trade.SyntheticTime = block.VegaTime
		trade.VegaTime = block.VegaTime
		blockTimes[trade.ID.String()] = block.VegaTime
		err := ts.Add(&trade)
		require.NoError(t, err)
		time.Sleep(time.Microsecond * 100)
	}

	_, err := ts.Flush(ctx)
	require.NoError(t, err)
}

func testTradesCursorPaginationByMarketNoCursor(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, entities.PartyID{}, entities.OrderID{}, pagination)
	require.NoError(t, err)

	assert.Len(t, got, 4)
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, uint64(1), got[0].Size)
	assert.Equal(t, "7a797e0e", got[3].ID.String())
	assert.Equal(t, uint64(4), got[3].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, "7a797e0e", got[3].ID.String())
}

func testTradesCursorPaginationByPartyNoMarketNoCursor(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	partyID := entities.NewPartyID("dabbad00")
	got, pageInfo, err := ts.List(ctx, entities.MarketID{}, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 5)

	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, uint64(1), got[0].Size)
	assert.Equal(t, "7bb2356e", got[4].ID.String())
	assert.Equal(t, uint64(5), got[4].Size)

	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, "7bb2356e", got[4].ID.String())
}

func testTradesCursorPaginationByPartyAndMarketNoCursor(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 4)
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, uint64(1), got[0].Size)
	assert.Equal(t, "7a797e0e", got[3].ID.String())
	assert.Equal(t, uint64(4), got[3].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, "7a797e0e", got[3].ID.String())
}

func testTradesCursorPaginationByMarketWithCursorFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, entities.PartyID{}, entities.OrderID{}, pagination)
	require.NoError(t, err)

	assert.Len(t, got, 2)
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, uint64(1), got[0].Size)
	assert.Equal(t, "44eea1bc", got[1].ID.String())
	assert.Equal(t, uint64(2), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, "44eea1bc", got[1].ID.String())
}

func testTradesCursorPaginationByPartyWithCursorNoMarketFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	got, pageInfo, err := ts.List(ctx, entities.MarketID{}, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, uint64(1), got[0].Size)
	assert.Equal(t, "44eea1bc", got[1].ID.String())
	assert.Equal(t, uint64(2), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, "44eea1bc", got[1].ID.String())
}

func testTradesCursorPaginationByPartyAndMarketWithCursorFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, uint64(1), got[0].Size)
	assert.Equal(t, "44eea1bc", got[1].ID.String())
	assert.Equal(t, uint64(2), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, "44eea1bc", got[1].ID.String())
}

func testTradesCursorPaginationByMarketWithCursorLast(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, entities.PartyID{}, entities.OrderID{}, pagination)
	require.NoError(t, err)

	assert.Len(t, got, 2)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, uint64(3), got[0].Size)
	assert.Equal(t, "7a797e0e", got[1].ID.String())
	assert.Equal(t, uint64(4), got[1].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, "7a797e0e", got[1].ID.String())
}

func testTradesCursorPaginationByPartyWithCursorNoMarketLast(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	got, pageInfo, err := ts.List(ctx, entities.MarketID{}, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, uint64(4), got[0].Size)
	assert.Equal(t, "7bb2356e", got[1].ID.String())
	assert.Equal(t, uint64(5), got[1].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, "7bb2356e", got[1].ID.String())
}

func testTradesCursorPaginationByPartyAndMarketWithCursorLast(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	last := int32(2)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, uint64(3), got[0].Size)
	assert.Equal(t, "7a797e0e", got[1].ID.String())
	assert.Equal(t, uint64(4), got[1].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, "7a797e0e", got[1].ID.String())
}

func testTradesCursorPaginationByMarketWithCursorForward(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(blockTimes["02a16077"].Format(time.RFC3339Nano)).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, entities.PartyID{}, entities.OrderID{}, pagination)
	require.NoError(t, err)

	assert.Len(t, got, 2)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, uint64(2), got[0].Size)
	assert.Equal(t, "65be62cd", got[1].ID.String())
	assert.Equal(t, uint64(3), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, "65be62cd", got[1].ID.String())
}

func testTradesCursorPaginationByPartyWithCursorNoMarketForward(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(blockTimes["44eea1bc"].Format(time.RFC3339Nano)).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	got, pageInfo, err := ts.List(ctx, entities.MarketID{}, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, uint64(3), got[0].Size)
	assert.Equal(t, "7a797e0e", got[1].ID.String())
	assert.Equal(t, uint64(4), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, "7a797e0e", got[1].ID.String())
}

func testTradesCursorPaginationByPartyAndMarketWithCursorForward(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(blockTimes["02a16077"].Format(time.RFC3339Nano)).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, uint64(2), got[0].Size)
	assert.Equal(t, "65be62cd", got[1].ID.String())
	assert.Equal(t, uint64(3), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, "65be62cd", got[1].ID.String())
}

func testTradesCursorPaginationByMarketWithCursorBackward(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	before := entities.NewCursor(blockTimes["7a797e0e"].Format(time.RFC3339Nano)).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, entities.PartyID{}, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, uint64(2), got[0].Size)
	assert.Equal(t, "65be62cd", got[1].ID.String())
	assert.Equal(t, uint64(3), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, "65be62cd", got[1].ID.String())
}

func testTradesCursorPaginationByPartyWithCursorNoMarketBackward(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	before := entities.NewCursor(blockTimes["7bb2356e"].Format(time.RFC3339Nano)).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	got, pageInfo, err := ts.List(ctx, entities.MarketID{}, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, uint64(3), got[0].Size)
	assert.Equal(t, "7a797e0e", got[1].ID.String())
	assert.Equal(t, uint64(4), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, "7a797e0e", got[1].ID.String())
}

func testTradesCursorPaginationByPartyAndMarketWithCursorBackward(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	before := entities.NewCursor(blockTimes["7a797e0e"].Format(time.RFC3339Nano)).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, uint64(2), got[0].Size)
	assert.Equal(t, "65be62cd", got[1].ID.String())
	assert.Equal(t, uint64(3), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, "65be62cd", got[1].ID.String())
}

// Newest First
func testTradesCursorPaginationByMarketNoCursorNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, entities.PartyID{}, entities.OrderID{}, pagination)
	require.NoError(t, err)

	assert.Len(t, got, 4)
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, uint64(4), got[0].Size)
	assert.Equal(t, "02a16077", got[3].ID.String())
	assert.Equal(t, uint64(1), got[3].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, "02a16077", got[3].ID.String())
}

func testTradesCursorPaginationByPartyNoMarketNoCursorNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	partyID := entities.NewPartyID("dabbad00")
	got, pageInfo, err := ts.List(ctx, entities.MarketID{}, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 5)

	assert.Equal(t, "7bb2356e", got[0].ID.String())
	assert.Equal(t, uint64(5), got[0].Size)
	assert.Equal(t, "02a16077", got[4].ID.String())
	assert.Equal(t, uint64(1), got[4].Size)

	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "7bb2356e", got[0].ID.String())
	assert.Equal(t, "02a16077", got[4].ID.String())
}

func testTradesCursorPaginationByPartyAndMarketNoCursorNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 4)
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, uint64(4), got[0].Size)
	assert.Equal(t, "02a16077", got[3].ID.String())
	assert.Equal(t, uint64(1), got[3].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, "02a16077", got[3].ID.String())
}

func testTradesCursorPaginationByMarketWithCursorFirstNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, entities.PartyID{}, entities.OrderID{}, pagination)
	require.NoError(t, err)

	assert.Len(t, got, 2)
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, uint64(4), got[0].Size)
	assert.Equal(t, "65be62cd", got[1].ID.String())
	assert.Equal(t, uint64(3), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, "65be62cd", got[1].ID.String())
}

func testTradesCursorPaginationByPartyWithCursorNoMarketFirstNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	got, pageInfo, err := ts.List(ctx, entities.MarketID{}, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "7bb2356e", got[0].ID.String())
	assert.Equal(t, uint64(5), got[0].Size)
	assert.Equal(t, "7a797e0e", got[1].ID.String())
	assert.Equal(t, uint64(4), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "7bb2356e", got[0].ID.String())
	assert.Equal(t, "7a797e0e", got[1].ID.String())
}

func testTradesCursorPaginationByPartyAndMarketWithCursorFirstNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	first := int32(2)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, uint64(4), got[0].Size)
	assert.Equal(t, "65be62cd", got[1].ID.String())
	assert.Equal(t, uint64(3), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, "65be62cd", got[1].ID.String())
}

func testTradesCursorPaginationByMarketWithCursorLastNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, entities.PartyID{}, entities.OrderID{}, pagination)
	require.NoError(t, err)

	assert.Len(t, got, 2)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, uint64(2), got[0].Size)
	assert.Equal(t, "02a16077", got[1].ID.String())
	assert.Equal(t, uint64(1), got[1].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, "02a16077", got[1].ID.String())
}

func testTradesCursorPaginationByPartyWithCursorNoMarketLastNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	got, pageInfo, err := ts.List(ctx, entities.MarketID{}, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, uint64(2), got[0].Size)
	assert.Equal(t, "02a16077", got[1].ID.String())
	assert.Equal(t, uint64(1), got[1].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, "02a16077", got[1].ID.String())
}

func testTradesCursorPaginationByPartyAndMarketWithCursorLastNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, uint64(2), got[0].Size)
	assert.Equal(t, "02a16077", got[1].ID.String())
	assert.Equal(t, uint64(1), got[1].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, "02a16077", got[1].ID.String())
}

func testTradesCursorPaginationByMarketWithCursorForwardNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(blockTimes["7a797e0e"].Format(time.RFC3339Nano)).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, entities.PartyID{}, entities.OrderID{}, pagination)
	require.NoError(t, err)

	assert.Len(t, got, 2)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, uint64(3), got[0].Size)
	assert.Equal(t, "44eea1bc", got[1].ID.String())
	assert.Equal(t, uint64(2), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, "44eea1bc", got[1].ID.String())
}

func testTradesCursorPaginationByPartyWithCursorNoMarketForwardNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(blockTimes["7a797e0e"].Format(time.RFC3339Nano)).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	got, pageInfo, err := ts.List(ctx, entities.MarketID{}, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, uint64(3), got[0].Size)
	assert.Equal(t, "44eea1bc", got[1].ID.String())
	assert.Equal(t, uint64(2), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, "44eea1bc", got[1].ID.String())
}

func testTradesCursorPaginationByPartyAndMarketWithCursorForwardNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(blockTimes["7a797e0e"].Format(time.RFC3339Nano)).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, uint64(3), got[0].Size)
	assert.Equal(t, "44eea1bc", got[1].ID.String())
	assert.Equal(t, uint64(2), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, "44eea1bc", got[1].ID.String())
}

func testTradesCursorPaginationByMarketWithCursorBackwardNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	before := entities.NewCursor(blockTimes["02a16077"].Format(time.RFC3339Nano)).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, entities.PartyID{}, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, uint64(3), got[0].Size)
	assert.Equal(t, "44eea1bc", got[1].ID.String())
	assert.Equal(t, uint64(2), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, "44eea1bc", got[1].ID.String())
}

func testTradesCursorPaginationByPartyWithCursorNoMarketBackwardNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	before := entities.NewCursor(blockTimes["44eea1bc"].Format(time.RFC3339Nano)).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	got, pageInfo, err := ts.List(ctx, entities.MarketID{}, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, uint64(4), got[0].Size)
	assert.Equal(t, "65be62cd", got[1].ID.String())
	assert.Equal(t, uint64(3), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, "65be62cd", got[1].ID.String())
}

func testTradesCursorPaginationByPartyAndMarketWithCursorBackwardNewestFirst(t *testing.T) {
	bs, ts, _, teardown := setupTradesTest(t)
	t.Logf("DB Port: %d", testDBPort)

	defer teardown(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	before := entities.NewCursor(blockTimes["02a16077"].Format(time.RFC3339Nano)).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	partyID := entities.NewPartyID("dabbad00")
	marketID := entities.NewMarketID("deadbeef")
	got, pageInfo, err := ts.List(ctx, marketID, partyID, entities.OrderID{}, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, uint64(3), got[0].Size)
	assert.Equal(t, "44eea1bc", got[1].ID.String())
	assert.Equal(t, uint64(2), got[1].Size)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, "44eea1bc", got[1].ID.String())
}
