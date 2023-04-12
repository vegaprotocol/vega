// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

	types "code.vegaprotocol.io/vega/protos/vega"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"github.com/stretchr/testify/assert"
)

const (
	testMarket = "b4376d805a888548baabfae74ef6f4fa4680dc9718bab355fa7191715de4fafe"
	testPartyA = "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"
	testPartyB = "521127F24B1FA40311BA2FB3F6977310346346604B275DB7B767B04240A5A5C3"
	orderAId   = "787B72CB5DD7A5EA869E49F361CF957DF747F849B4ACE88ABC6DA0F9C450AFDD"
	orderBId   = "83dc82be23c77daec384a239143f07f83c667acf60d734745b023c6567e7b57b"

	tradeID1 = "0bd678723c33b059638953e0904d2ddbd78c2be72ab25a8753a622911c2d9c78"
	tradeID2 = "af2bb48edd738353fcd7a2b6cea4821dd2382ec95497954535278dfbfff7b5b5"
	tradeID3 = "3d4ed10064b7cedbc8a37316f7329f853c9588b6a55006ffb8bec3f1a4ccc88e"
	tradeID4 = "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"
	tradeID5 = "8b6be1a03cc4d529f682887a78b66e6879d17f81e2b37356ca0acbc5d5886eb8"
	tradeID6 = "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"
)

func TestStorageGetByTxHash(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	tradeStore := sqlstore.NewTrades(connectionSource)

	insertedTrades := insertTestData(t, ctx, tradeStore)

	trades, err := tradeStore.GetByTxHash(ctx, insertedTrades[0].TxHash)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, insertedTrades[0].ID.String(), trades[0].ID.String())
	assert.Equal(t, insertedTrades[0].TxHash, trades[0].TxHash)

	trades, err = tradeStore.GetByTxHash(ctx, insertedTrades[2].TxHash)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, insertedTrades[2].ID.String(), trades[0].ID.String())
	assert.Equal(t, insertedTrades[2].TxHash, trades[0].TxHash)
}

func insertTestData(t *testing.T, ctx context.Context, tradeStore *sqlstore.Trades) []entities.Trade {
	t.Helper()

	// Insert some trades
	bs := sqlstore.NewBlocks(connectionSource)
	now := time.Now()
	block1 := addTestBlockForTime(t, ctx, bs, now)
	block2 := addTestBlockForTime(t, ctx, bs, now.Add(time.Second))

	trade1 := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        tradeID1,
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
		Id:        tradeID2,
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
		Id:        tradeID3,
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
		Id:        tradeID4,
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
		Id:        tradeID5,
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
		Id:        tradeID6,
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

	inserted := []entities.Trade{}
	var seqNum uint64
	vegaTime := block1.VegaTime
	for _, proto := range protos {
		if seqNum == 3 {
			seqNum = 0
			vegaTime = block2.VegaTime
		}
		trade, err := entities.TradeFromProto(&proto, generateTxHash(), vegaTime, seqNum)
		if err != nil {
			t.Fatalf("failed to get trade from proto:%s", err)
		}
		err = tradeStore.Add(trade)
		if err != nil {
			t.Fatalf("failed to add trade:%s", err)
		}
		seqNum++

		inserted = append(inserted, *trade)
	}

	tradeStore.Flush(ctx)
	return inserted
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

	t.Run("Should return all trades between dates for a given market when no cursor is given", testTradesCursorPaginationBetweenDatesByMarketNoCursor)
	t.Run("Should return all trades between dates for a given market when no cursor is given - newest first", testTradesCursorPaginationBetweenDatesByMarketNoCursorNewestFirst)
	t.Run("Should return the last page of trades between dates for a given market when a last cursor is set", testTradesCursorPaginationBetweenDatesByMarketWithCursorLast)
	t.Run("Should return the page of trades between dates for a given market when a first and after cursor is set", testTradesCursorPaginationBetweenDatesByMarketWithCursorForward)
}

func setupTradesTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.Trades) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ts := sqlstore.NewTrades(connectionSource)
	return bs, ts
}

func populateTestTrades(ctx context.Context, t *testing.T, bs *sqlstore.Blocks, ts *sqlstore.Trades, blockTimes map[string]time.Time) {
	t.Helper()

	trades := []entities.Trade{
		{
			SeqNum:    1,
			ID:        entities.TradeID("02a16077"),
			MarketID:  entities.MarketID("deadbeef"),
			Price:     decimal.NewFromFloat(1.0),
			Size:      1,
			Buyer:     entities.PartyID("dabbad00"),
			Seller:    entities.PartyID("facefeed"),
			BuyOrder:  entities.OrderID("02a16077"),
			SellOrder: entities.OrderID("fb1528a5"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    2,
			ID:        entities.TradeID("44eea1bc"),
			MarketID:  entities.MarketID("deadbeef"),
			Price:     decimal.NewFromFloat(2.0),
			Size:      2,
			Buyer:     entities.PartyID("dabbad00"),
			Seller:    entities.PartyID("facefeed"),
			BuyOrder:  entities.OrderID("44eea1bc"),
			SellOrder: entities.OrderID("da8d1803"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    3,
			ID:        entities.TradeID("65be62cd"),
			MarketID:  entities.MarketID("deadbeef"),
			Price:     decimal.NewFromFloat(3.0),
			Size:      3,
			Buyer:     entities.PartyID("dabbad00"),
			Seller:    entities.PartyID("facefeed"),
			BuyOrder:  entities.OrderID("65be62cd"),
			SellOrder: entities.OrderID("c8744329"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    4,
			ID:        entities.TradeID("7a797e0e"),
			MarketID:  entities.MarketID("deadbeef"),
			Price:     decimal.NewFromFloat(4.0),
			Size:      4,
			Buyer:     entities.PartyID("dabbad00"),
			Seller:    entities.PartyID("facefeed"),
			BuyOrder:  entities.OrderID("7a797e0e"),
			SellOrder: entities.OrderID("c612300d"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    5,
			ID:        entities.TradeID("7bb2356e"),
			MarketID:  entities.MarketID("cafed00d"),
			Price:     decimal.NewFromFloat(5.0),
			Size:      5,
			Buyer:     entities.PartyID("dabbad00"),
			Seller:    entities.PartyID("facefeed"),
			BuyOrder:  entities.OrderID("7bb2356e"),
			SellOrder: entities.OrderID("b7c84b8e"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    6,
			ID:        entities.TradeID("b7c84b8e"),
			MarketID:  entities.MarketID("cafed00d"),
			Price:     decimal.NewFromFloat(6.0),
			Size:      6,
			Buyer:     entities.PartyID("d0d0caca"),
			Seller:    entities.PartyID("decafbad"),
			BuyOrder:  entities.OrderID("b7c84b8e"),
			SellOrder: entities.OrderID("7bb2356e"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    7,
			ID:        entities.TradeID("c612300d"),
			MarketID:  entities.MarketID("cafed00d"),
			Price:     decimal.NewFromFloat(7.0),
			Size:      7,
			Buyer:     entities.PartyID("d0d0caca"),
			Seller:    entities.PartyID("decafbad"),
			BuyOrder:  entities.OrderID("c612300d"),
			SellOrder: entities.OrderID("7a797e0e"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    8,
			ID:        entities.TradeID("c8744329"),
			MarketID:  entities.MarketID("cafed00d"),
			Price:     decimal.NewFromFloat(8.0),
			Size:      8,
			Buyer:     entities.PartyID("d0d0caca"),
			Seller:    entities.PartyID("decafbad"),
			BuyOrder:  entities.OrderID("c8744329"),
			SellOrder: entities.OrderID("65be62cd"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    9,
			ID:        entities.TradeID("da8d1803"),
			MarketID:  entities.MarketID("deadbaad"),
			Price:     decimal.NewFromFloat(9.0),
			Size:      9,
			Buyer:     entities.PartyID("baadf00d"),
			Seller:    entities.PartyID("0d15ea5e"),
			BuyOrder:  entities.OrderID("da8d1803"),
			SellOrder: entities.OrderID("44eea1bc"),
			Type:      entities.TradeTypeDefault,
		},
		{
			SeqNum:    10,
			ID:        entities.TradeID("fb1528a5"),
			MarketID:  entities.MarketID("deadbaad"),
			Price:     decimal.NewFromFloat(10.0),
			Size:      10,
			Buyer:     entities.PartyID("baadf00d"),
			Seller:    entities.PartyID("0d15ea5e"),
			BuyOrder:  entities.OrderID("fb1528a5"),
			SellOrder: entities.OrderID("02a16077"),
			Type:      entities.TradeTypeDefault,
		},
	}

	for _, td := range trades {
		trade := td
		block := addTestBlock(t, ctx, bs)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	partyID := []entities.PartyID{"dabbad00"}
	got, pageInfo, err := ts.List(ctx, nil, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	got, pageInfo, err := ts.List(ctx, nil, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	got, pageInfo, err := ts.List(ctx, nil, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	last := int32(2)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["02a16077"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["44eea1bc"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	got, pageInfo, err := ts.List(ctx, nil, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["02a16077"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	before := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["7a797e0e"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	before := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["7bb2356e"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	got, pageInfo, err := ts.List(ctx, nil, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	before := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["7a797e0e"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, partyID, nil, pagination, entities.DateRange{})
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

// Newest First.
func testTradesCursorPaginationByMarketNoCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	partyID := []entities.PartyID{"dabbad00"}
	got, pageInfo, err := ts.List(ctx, nil, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	got, pageInfo, err := ts.List(ctx, nil, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	first := int32(2)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	got, pageInfo, err := ts.List(ctx, nil, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["7a797e0e"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["7a797e0e"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	got, pageInfo, err := ts.List(ctx, nil, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["7a797e0e"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	before := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["02a16077"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	before := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["44eea1bc"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	got, pageInfo, err := ts.List(ctx, nil, partyID, nil, pagination, entities.DateRange{})
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	before := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["02a16077"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	partyID := []entities.PartyID{"dabbad00"}
	marketID := []entities.MarketID{"deadbeef"}
	got, pageInfo, err := ts.List(ctx, marketID, partyID, nil, pagination, entities.DateRange{})
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

func testTradesCursorPaginationBetweenDatesByMarketNoCursor(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	marketID := []entities.MarketID{"deadbeef"}
	startDate := blockTimes["44eea1bc"]
	endDate := blockTimes["7a797e0e"]
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})
	require.NoError(t, err)

	assert.Len(t, got, 2)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, uint64(2), got[0].Size)
	assert.Equal(t, "65be62cd", got[1].ID.String())
	assert.Equal(t, uint64(3), got[1].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
}

func testTradesCursorPaginationBetweenDatesByMarketNoCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	marketID := []entities.MarketID{"deadbeef"}
	startDate := blockTimes["44eea1bc"]
	endDate := blockTimes["7a797e0e"]
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})
	require.NoError(t, err)

	assert.Len(t, got, 2)
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, uint64(3), got[0].Size)
	assert.Equal(t, "44eea1bc", got[1].ID.String())
	assert.Equal(t, uint64(2), got[1].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
}

func testTradesCursorPaginationBetweenDatesByMarketWithCursorLast(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	last := int32(2)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	marketID := []entities.MarketID{"deadbeef"}
	startDate := blockTimes["02a16077"]
	endDate := blockTimes["7a797e0e"]
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})
	require.NoError(t, err)

	assert.Len(t, got, 2)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, uint64(2), got[0].Size)
	assert.Equal(t, "65be62cd", got[1].ID.String())
	assert.Equal(t, uint64(3), got[1].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
}

func testTradesCursorPaginationBetweenDatesByMarketWithCursorForward(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ts := setupTradesTest(t)
	blockTimes := make(map[string]time.Time)
	populateTestTrades(ctx, t, bs, ts, blockTimes)
	first := int32(2)
	after := entities.NewCursor(entities.TradeCursor{SyntheticTime: blockTimes["02a16077"]}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	marketID := []entities.MarketID{"deadbeef"}
	startDate := blockTimes["02a16077"]
	endDate := blockTimes["7a797e0e"]
	got, pageInfo, err := ts.List(ctx, marketID, nil, nil, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})
	require.NoError(t, err)

	assert.Len(t, got, 2)
	assert.Equal(t, "44eea1bc", got[0].ID.String())
	assert.Equal(t, uint64(2), got[0].Size)
	assert.Equal(t, "65be62cd", got[1].ID.String())
	assert.Equal(t, uint64(3), got[1].Size)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
}
