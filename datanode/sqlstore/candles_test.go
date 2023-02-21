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
	"strconv"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"github.com/stretchr/testify/require"

	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
)

const (
	StartTime            = 1649116800
	totalBlocks          = 1000
	tradesPerBlock       = 5
	startPrice           = 1
	size                 = 10
	priceIncrement       = 1
	blockIntervalSeconds = 10
	blockIntervalDur     = time.Duration(blockIntervalSeconds) * time.Second
)

func TestGetExistingCandles(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	candleStore := sqlstore.NewCandles(ctx, connectionSource, candlesv2.NewDefaultConfig().CandleStore)

	candles, err := candleStore.GetCandlesForMarket(ctx, testMarket)
	if err != nil {
		t.Fatalf("failed to get candles for market:%s", err)
	}

	defaultCandles := "block,1 minute,5 minutes,15 minutes,1 hour,6 hours,1 day"
	intervals := strings.Split(defaultCandles, ",")
	assert.Equal(t, len(intervals), len(candles))

	for _, interval := range intervals {
		candleID := candles[interval]
		exists, _ := candleStore.CandleExists(ctx, candleID)
		assert.True(t, exists)
	}
}

func TestCandlesPagination(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	candleStore := sqlstore.NewCandles(ctx, connectionSource, candlesv2.NewDefaultConfig().CandleStore)

	tradeStore := sqlstore.NewTrades(connectionSource)

	startTime := time.Unix(StartTime, 0)
	insertCandlesTestData(t, ctx, tradeStore, startTime, totalBlocks, tradesPerBlock, startPrice, priceIncrement, size, blockIntervalDur)

	_, candleID, _ := candleStore.GetCandleIDForIntervalAndMarket(ctx, "1 Minute", testMarket)
	first := int32(10)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	candles, _, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, nil,
		nil, pagination)
	if err != nil {
		t.Fatalf("failed to get candles with pagination:%s", err)
	}

	assert.Equal(t, 10, len(candles))
	lastCandle := candles[9]

	first = int32(5)
	after := candles[8].Cursor().Encode()

	pagination, _ = entities.NewCursorPagination(&first, &after, nil, nil, false)

	candles, _, err = candleStore.GetCandleDataForTimeSpan(ctx, candleID, nil,
		nil, pagination)

	if err != nil {
		t.Fatalf("failed to get candles with pagination:%s", err)
	}

	assert.Equal(t, 5, len(candles))
	assert.Equal(t, lastCandle, candles[0])
}

func TestCandlesGetForEmptyInterval(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	candleStore := sqlstore.NewCandles(ctx, connectionSource, candlesv2.NewDefaultConfig().CandleStore)
	tradeStore := sqlstore.NewTrades(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	startTime := time.Unix(StartTime, 0)
	block := addTestBlockForTime(t, ctx, bs, startTime)

	insertTestTrade(t, ctx, tradeStore, 1, 10, block, 0)
	insertTestTrade(t, ctx, tradeStore, 2, 10, block, 3)

	nextTime := time.Unix(StartTime, 0).Add(10 * time.Minute)
	block = addTestBlockForTime(t, ctx, bs, nextTime)
	insertTestTrade(t, ctx, tradeStore, 3, 20, block, 0)
	insertTestTrade(t, ctx, tradeStore, 4, 20, block, 5)

	_, candleID, err := candleStore.GetCandleIDForIntervalAndMarket(ctx, "1 Minute", testMarket)
	if err != nil {
		t.Fatalf("getting existing candleDescriptor id:%s", err)
	}

	pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil, false)
	candles, _, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, &startTime,
		nil, pagination)
	if err != nil {
		t.Fatalf("failed to get candles:%s", err)
	}

	assert.Equal(t, 2, len(candles))

	firstCandle := createCandle(startTime,
		startTime.Add(3*time.Microsecond), 1, 2, 2, 1, 20)
	assert.Equal(t, firstCandle, candles[0])

	secondCandle := createCandle(startTime.Add(10*time.Minute),
		startTime.Add(10*time.Minute).Add(5*time.Microsecond), 3, 4, 4, 3, 40)
	assert.Equal(t, secondCandle, candles[1])
}

func TestCandlesGetLatest(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	candleStore := sqlstore.NewCandles(ctx, connectionSource, candlesv2.NewDefaultConfig().CandleStore)
	tradeStore := sqlstore.NewTrades(connectionSource)

	startTime := time.Unix(StartTime, 0)
	insertCandlesTestData(t, ctx, tradeStore, startTime, 90, 3, startPrice, priceIncrement, size,
		1*time.Second)

	last := int32(1)
	pagination, _ := entities.NewCursorPagination(nil, nil, &last, nil, false)
	_, candleID, _ := candleStore.GetCandleIDForIntervalAndMarket(ctx, "1 Minute", testMarket)
	candles, _, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, &startTime,
		nil, pagination)
	if err != nil {
		t.Fatalf("failed to get candles:%s", err)
	}

	assert.Equal(t, 1, len(candles))

	lastCandle := createCandle(startTime.Add(60*time.Second),
		startTime.Add(89*time.Second).Add(2*time.Microsecond), 181, 270, 270, 181,
		900)
	assert.Equal(t, lastCandle, candles[0])
}

func TestCandlesGetForDifferentIntervalAndTimeBounds(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	candleStore := sqlstore.NewCandles(ctx, connectionSource, candlesv2.NewDefaultConfig().CandleStore)
	tradeStore := sqlstore.NewTrades(connectionSource)

	startTime := time.Unix(StartTime, 0)
	insertCandlesTestData(t, ctx, tradeStore, startTime, totalBlocks, tradesPerBlock, startPrice, priceIncrement, size, blockIntervalDur)

	testInterval(t, ctx, startTime, nil, nil, candleStore, "1 Minute", 60)
	testInterval(t, ctx, startTime, nil, nil, candleStore, "5 Minutes", 300)
	testInterval(t, ctx, startTime, nil, nil, candleStore, "15 Minutes", 900)
	testInterval(t, ctx, startTime, nil, nil, candleStore, "1 hour", 3600)

	from := startTime.Add(5 * time.Minute)
	to := startTime.Add(35 * time.Minute)

	testInterval(t, ctx, startTime, &from, &to, candleStore, "1 Minute", 60)
	testInterval(t, ctx, startTime, &from, &to, candleStore, "5 Minutes", 300)

	testInterval(t, ctx, startTime, nil, &to, candleStore, "1 Minute", 60)
	testInterval(t, ctx, startTime, nil, &to, candleStore, "5 Minutes", 300)

	testInterval(t, ctx, startTime, &from, nil, candleStore, "1 Minute", 60)
	testInterval(t, ctx, startTime, &from, nil, candleStore, "5 Minutes", 300)
}

func testInterval(t *testing.T, ctx context.Context, tradeDataStartTime time.Time, fromTime *time.Time, toTime *time.Time, candleStore *sqlstore.Candles, interval string,
	intervalSeconds int,
) {
	t.Helper()
	intervalDur := time.Duration(intervalSeconds) * time.Second

	pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil, false)
	// entities.OffsetPagination{}
	_, candleID, _ := candleStore.GetCandleIDForIntervalAndMarket(ctx, interval, testMarket)
	candles, _, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, fromTime,
		toTime, pagination)
	if err != nil {
		t.Fatalf("failed to get candles:%s", err)
	}

	tradeDataTimeSpanSeconds := totalBlocks * blockIntervalSeconds
	tradeDataEndTime := tradeDataStartTime.Add(time.Duration(tradeDataTimeSpanSeconds) * time.Second)

	var candlesStartTime time.Time
	if fromTime != nil && fromTime.After(tradeDataStartTime) {
		candlesStartTime = *fromTime
	} else {
		candlesStartTime = tradeDataStartTime
	}

	var candlesEndTime time.Time
	if toTime != nil && toTime.Before(tradeDataEndTime) {
		candlesEndTime = *toTime
	} else {
		candlesEndTime = tradeDataEndTime
	}

	candleSpan := candlesEndTime.Sub(candlesStartTime)

	expectedNumCandles := int(candleSpan.Seconds() / float64(intervalSeconds))

	if toTime == nil {
		expectedNumCandles = expectedNumCandles + 1
	}

	assert.Equal(t, expectedNumCandles, len(candles))

	blocksPerInterval := intervalSeconds / blockIntervalSeconds

	skippedTrades := int(candlesStartTime.Sub(tradeDataStartTime).Seconds()/blockIntervalSeconds) * tradesPerBlock

	for idx := 0; idx < expectedNumCandles-1; idx++ {
		periodStart := candlesStartTime.Add(time.Duration(idx) * intervalDur)
		tradesAtOpen := skippedTrades + idx*tradesPerBlock*blocksPerInterval
		tradesAtClose := skippedTrades + (idx+1)*tradesPerBlock*blocksPerInterval
		expectedVolume := tradesPerBlock * blocksPerInterval * size
		candle := candles[idx]
		expectedCandle := createCandle(periodStart,
			periodStart.Add(time.Duration(tradesPerBlock-1)*time.Microsecond).Add(intervalDur).Add(-1*blockIntervalDur),
			startPrice+tradesAtOpen, tradesAtClose, tradesAtClose, startPrice+tradesAtOpen,
			expectedVolume)
		assert.Equal(t, expectedCandle, candle)
	}
}

func createCandle(periodStart time.Time, lastUpdate time.Time, open int, close int, high int, low int, volume int) entities.Candle {
	return entities.Candle{
		PeriodStart:        periodStart,
		LastUpdateInPeriod: lastUpdate,
		Open:               decimal.NewFromInt(int64(open)),
		Close:              decimal.NewFromInt(int64(close)),
		High:               decimal.NewFromInt(int64(high)),
		Low:                decimal.NewFromInt(int64(low)),
		Volume:             uint64(volume),
	}
}

//nolint:unparam
func insertCandlesTestData(t *testing.T, ctx context.Context, tradeStore *sqlstore.Trades, startTime time.Time, numBlocks int,
	tradePerBlock int, startPrice int, priceIncrement int, size int, blockIntervalDur time.Duration,
) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)

	var blocks []entities.Block
	for i := 0; i < numBlocks; i++ {
		blocks = append(blocks, addTestBlockForTime(t, ctx, bs, startTime.Add(time.Duration(i)*blockIntervalDur)))
	}

	for _, block := range blocks {
		for seqNum := 0; seqNum < tradePerBlock; seqNum++ {
			trade := createTestTrade(t, startPrice, size, block, seqNum)
			err := tradeStore.Add(trade)
			if err != nil {
				t.Fatalf("failed to add trade to store:%s", err)
			}
			startPrice = startPrice + priceIncrement
		}
	}

	_, err := tradeStore.Flush(ctx)
	assert.NoError(t, err)
}

func insertTestTrade(t *testing.T, ctx context.Context, tradeStore *sqlstore.Trades, price int, size int, block entities.Block, seqNum int) {
	t.Helper()
	trade := createTestTrade(t, price, size, block, seqNum)
	insertTrade(t, ctx, tradeStore, trade)
}

func insertTrade(t *testing.T, ctx context.Context, tradeStore *sqlstore.Trades, trade *entities.Trade) *entities.Trade {
	t.Helper()
	err := tradeStore.Add(trade)
	tradeStore.Flush(ctx)
	if err != nil {
		t.Fatalf("failed to add trade to store:%s", err)
	}

	return trade
}

func createTestTrade(t *testing.T, price int, size int, block entities.Block, seqNum int) *entities.Trade {
	t.Helper()
	proto := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        helpers.GenerateID(),
		Price:     strconv.Itoa(price),
		Size:      uint64(size),
		MarketId:  testMarket,
		Buyer:     helpers.GenerateID(),
		Seller:    helpers.GenerateID(),
		Aggressor: types.Side_SIDE_SELL,
		BuyOrder:  helpers.GenerateID(),
		SellOrder: helpers.GenerateID(),
	}

	trade, err := entities.TradeFromProto(proto, generateTxHash(), block.VegaTime, uint64(seqNum))
	if err != nil {
		t.Fatalf("failed to create trade from proto:%s", err)
	}
	return trade
}

func TestCandlesCursorPagination(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	candleStore := sqlstore.NewCandles(ctx, connectionSource, candlesv2.NewDefaultConfig().CandleStore)
	tradeStore := sqlstore.NewTrades(connectionSource)

	startTime := time.Unix(StartTime, 0)
	insertCandlesTestData(t, ctx, tradeStore, startTime, totalBlocks, tradesPerBlock, startPrice, priceIncrement, size, blockIntervalDur)

	_, candleID, err := candleStore.GetCandleIDForIntervalAndMarket(ctx, "1 Minute", testMarket)
	if err != nil {
		t.Fatalf("getting existing candleDescriptor id:%s", err)
	}

	pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil, false)
	// retrieve all candles without pagination to use for test validation
	allCandles, _, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, &startTime,
		nil, pagination)
	if err != nil {
		t.Fatalf("failed to get candles:%s", err)
	}

	require.Equal(t, 167, len(allCandles))

	t.Run("should return the first candles when first is provided with no after", func(t *testing.T) {
		first := int32(10)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
		require.NoError(t, err)
		candles, pageInfo, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, &startTime, nil, pagination)
		require.NoError(t, err)
		require.Equal(t, 10, len(candles))
		assert.Equal(t, allCandles[0], candles[0])
		assert.Equal(t, allCandles[9], candles[9])
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     allCandles[0].Cursor().Encode(),
			EndCursor:       allCandles[9].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("should return the first candles when first is provided with no after - newest first", func(t *testing.T) {
		first := int32(10)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
		require.NoError(t, err)
		candles, pageInfo, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, &startTime, nil, pagination)
		require.NoError(t, err)
		lastIndex := len(allCandles) - 1
		require.Equal(t, 10, len(candles))
		assert.Equal(t, allCandles[lastIndex], candles[0])
		assert.Equal(t, allCandles[lastIndex-9], candles[9])
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     allCandles[lastIndex].Cursor().Encode(),
			EndCursor:       allCandles[lastIndex-9].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("should return the last page of candles when last is provided with no before", func(t *testing.T) {
		last := int32(10)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
		require.NoError(t, err)
		candles, pageInfo, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, &startTime, nil, pagination)
		require.NoError(t, err)
		require.Equal(t, 10, len(candles))
		assert.Equal(t, allCandles[157], candles[0])
		assert.Equal(t, allCandles[166], candles[9])
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     allCandles[157].Cursor().Encode(),
			EndCursor:       allCandles[166].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("should return the last page of candles when last is provided with no before - newest first", func(t *testing.T) {
		last := int32(10)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
		require.NoError(t, err)
		candles, pageInfo, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, &startTime, nil, pagination)
		require.NoError(t, err)
		require.Equal(t, 10, len(candles))
		assert.Equal(t, allCandles[9], candles[0])
		assert.Equal(t, allCandles[0], candles[9])
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     allCandles[9].Cursor().Encode(),
			EndCursor:       allCandles[0].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("should return the requested page of candles when first and after are provided", func(t *testing.T) {
		first := int32(10)
		after := allCandles[99].Cursor().Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
		require.NoError(t, err)
		candles, pageInfo, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, &startTime, nil, pagination)
		require.NoError(t, err)
		require.Equal(t, 10, len(candles))
		assert.Equal(t, allCandles[100], candles[0])
		assert.Equal(t, allCandles[109], candles[9])
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     allCandles[100].Cursor().Encode(),
			EndCursor:       allCandles[109].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("should return the requested page of candles when first and after are provided - newest first", func(t *testing.T) {
		first := int32(10)
		after := allCandles[99].Cursor().Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)
		candles, pageInfo, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, &startTime, nil, pagination)
		require.NoError(t, err)
		require.Equal(t, 10, len(candles))
		assert.Equal(t, allCandles[98], candles[0])
		assert.Equal(t, allCandles[89], candles[9])
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     allCandles[98].Cursor().Encode(),
			EndCursor:       allCandles[89].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested page of candles when last and before are provided", func(t *testing.T) {
		last := int32(10)
		before := allCandles[100].Cursor().Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
		require.NoError(t, err)
		candles, pageInfo, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, &startTime, nil, pagination)
		require.NoError(t, err)
		require.Equal(t, 10, len(candles))
		assert.Equal(t, allCandles[90], candles[0])
		assert.Equal(t, allCandles[99], candles[9])
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     allCandles[90].Cursor().Encode(),
			EndCursor:       allCandles[99].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested page of candles when last and before are provided - newest first", func(t *testing.T) {
		last := int32(10)
		before := allCandles[100].Cursor().Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
		require.NoError(t, err)
		candles, pageInfo, err := candleStore.GetCandleDataForTimeSpan(ctx, candleID, &startTime, nil, pagination)
		require.NoError(t, err)
		require.Equal(t, 10, len(candles))
		assert.Equal(t, allCandles[110], candles[0])
		assert.Equal(t, allCandles[101], candles[9])
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     allCandles[110].Cursor().Encode(),
			EndCursor:       allCandles[101].Cursor().Encode(),
		}, pageInfo)
	})
}
