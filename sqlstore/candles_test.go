package sqlstore_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/candlesv2"
	"github.com/stretchr/testify/require"

	types "code.vegaprotocol.io/protos/vega"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
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
	defer DeleteEverything()

	candleStore := sqlstore.NewCandles(context.Background(), connectionSource, candlesv2.NewDefaultConfig().CandleStore)

	candles, err := candleStore.GetCandlesForMarket(context.Background(), testMarket)
	if err != nil {
		t.Fatalf("failed to get candles for market:%s", err)
	}

	defaultCandles := "1 minute,5 minutes,15 minutes,1 hour,6 hours,1 day"
	intervals := strings.Split(defaultCandles, ",")
	assert.Equal(t, len(intervals), len(candles))

	for _, interval := range intervals {
		candleId := candles[interval]
		exists, _ := candleStore.CandleExists(context.Background(), candleId)
		assert.True(t, exists)
	}
}

func TestCandlesPagination(t *testing.T) {
	defer DeleteEverything()

	candleStore := sqlstore.NewCandles(context.Background(), connectionSource, candlesv2.NewDefaultConfig().CandleStore)

	tradeStore := sqlstore.NewTrades(connectionSource)

	startTime := time.Unix(StartTime, 0)
	insertCandlesTestData(t, tradeStore, startTime, totalBlocks, tradesPerBlock, startPrice, priceIncrement, size, blockIntervalDur)

	_, candleId, _ := candleStore.GetCandleIdForIntervalAndMarket(context.Background(), "1 Minute", testMarket)
	first := int32(10)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil)
	require.NoError(t, err)

	candles, _, err := candleStore.GetCandleDataForTimeSpan(context.Background(), candleId, nil,
		nil, pagination)

	if err != nil {
		t.Fatalf("failed to get candles with pagination:%s", err)
	}

	assert.Equal(t, 10, len(candles))
	lastCandle := candles[9]

	first = int32(5)
	after := entities.NewCursor(candles[8].PeriodStart.Format(time.RFC3339Nano)).Encode()

	pagination, _ = entities.NewCursorPagination(&first, &after, nil, nil)

	candles, _, err = candleStore.GetCandleDataForTimeSpan(context.Background(), candleId, nil,
		nil, pagination)

	if err != nil {
		t.Fatalf("failed to get candles with pagination:%s", err)
	}

	assert.Equal(t, 5, len(candles))
	assert.Equal(t, lastCandle, candles[0])
}

func TestCandlesGetForEmptyInterval(t *testing.T) {
	defer DeleteEverything()

	candleStore := sqlstore.NewCandles(context.Background(), connectionSource, candlesv2.NewDefaultConfig().CandleStore)
	tradeStore := sqlstore.NewTrades(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	startTime := time.Unix(StartTime, 0)
	block := addTestBlockForTime(t, bs, startTime)

	insertTestTrade(t, tradeStore, 1, 10, block, 0)
	insertTestTrade(t, tradeStore, 2, 10, block, 3)

	nextTime := time.Unix(StartTime, 0).Add(10 * time.Minute)
	block = addTestBlockForTime(t, bs, nextTime)
	insertTestTrade(t, tradeStore, 3, 20, block, 0)
	insertTestTrade(t, tradeStore, 4, 20, block, 5)

	_, candleId, err := candleStore.GetCandleIdForIntervalAndMarket(context.Background(), "1 Minute", testMarket)
	if err != nil {
		t.Fatalf("getting existing candleDescriptor id:%s", err)
	}

	pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil)
	candles, _, err := candleStore.GetCandleDataForTimeSpan(context.Background(), candleId, &startTime,
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
	defer DeleteEverything()

	candleStore := sqlstore.NewCandles(context.Background(), connectionSource, candlesv2.NewDefaultConfig().CandleStore)
	tradeStore := sqlstore.NewTrades(connectionSource)

	startTime := time.Unix(StartTime, 0)
	insertCandlesTestData(t, tradeStore, startTime, 90, 3, startPrice, priceIncrement, size,
		1*time.Second)

	last := int32(1)
	pagination, _ := entities.NewCursorPagination(nil, nil, &last, nil)
	_, candleId, _ := candleStore.GetCandleIdForIntervalAndMarket(context.Background(), "1 Minute", testMarket)
	candles, _, err := candleStore.GetCandleDataForTimeSpan(context.Background(), candleId, &startTime,
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
	defer DeleteEverything()

	candleStore := sqlstore.NewCandles(context.Background(), connectionSource, candlesv2.NewDefaultConfig().CandleStore)
	tradeStore := sqlstore.NewTrades(connectionSource)

	startTime := time.Unix(StartTime, 0)
	insertCandlesTestData(t, tradeStore, startTime, totalBlocks, tradesPerBlock, startPrice, priceIncrement, size, blockIntervalDur)

	testInterval(t, startTime, nil, nil, candleStore, "1 Minute", 60)
	testInterval(t, startTime, nil, nil, candleStore, "5 Minutes", 300)
	testInterval(t, startTime, nil, nil, candleStore, "15 Minutes", 900)
	testInterval(t, startTime, nil, nil, candleStore, "1 hour", 3600)

	from := startTime.Add(5 * time.Minute)
	to := startTime.Add(35 * time.Minute)

	testInterval(t, startTime, &from, &to, candleStore, "1 Minute", 60)
	testInterval(t, startTime, &from, &to, candleStore, "5 Minutes", 300)

	testInterval(t, startTime, nil, &to, candleStore, "1 Minute", 60)
	testInterval(t, startTime, nil, &to, candleStore, "5 Minutes", 300)

	testInterval(t, startTime, &from, nil, candleStore, "1 Minute", 60)
	testInterval(t, startTime, &from, nil, candleStore, "5 Minutes", 300)
}

func testInterval(t *testing.T, tradeDataStartTime time.Time, fromTime *time.Time, toTime *time.Time, candleStore *sqlstore.Candles, interval string,
	intervalSeconds int,
) {
	intervalDur := time.Duration(intervalSeconds) * time.Second

	pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil)
	//entities.OffsetPagination{}
	_, candleId, _ := candleStore.GetCandleIdForIntervalAndMarket(context.Background(), interval, testMarket)
	candles, _, err := candleStore.GetCandleDataForTimeSpan(context.Background(), candleId, fromTime,
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

	blocksPerInterval := int(intervalSeconds) / blockIntervalSeconds

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

func insertCandlesTestData(t *testing.T, tradeStore *sqlstore.Trades, startTime time.Time, numBlocks int,
	tradePerBlock int, startPrice int, priceIncrement int, size int, blockIntervalDur time.Duration,
) {
	bs := sqlstore.NewBlocks(connectionSource)

	var blocks []entities.Block
	for i := 0; i < numBlocks; i++ {
		blocks = append(blocks, addTestBlockForTime(t, bs, startTime.Add(time.Duration(i)*blockIntervalDur)))
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

	_, err := tradeStore.Flush(context.Background())
	assert.NoError(t, err)
}

func insertTestTrade(t *testing.T, tradeStore *sqlstore.Trades, price int, size int, block entities.Block, seqNum int) *entities.Trade {
	trade := createTestTrade(t, price, size, block, seqNum)
	return insertTrade(t, tradeStore, trade)
}

func insertTrade(t *testing.T, tradeStore *sqlstore.Trades, trade *entities.Trade) *entities.Trade {
	err := tradeStore.Add(trade)
	tradeStore.Flush(context.Background())
	if err != nil {
		t.Fatalf("failed to add trade to store:%s", err)
	}

	return trade
}

func createTestTrade(t *testing.T, price int, size int, block entities.Block, seqNum int) *entities.Trade {
	proto := &types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        generateID(),
		Price:     strconv.Itoa(price),
		Size:      uint64(size),
		MarketId:  testMarket,
		Buyer:     generateID(),
		Seller:    generateID(),
		Aggressor: types.Side_SIDE_SELL,
		BuyOrder:  generateID(),
		SellOrder: generateID(),
	}

	trade, err := entities.TradeFromProto(proto, block.VegaTime, uint64(seqNum))
	if err != nil {
		t.Fatalf("failed to create trade from proto:%s", err)
	}
	return trade
}

func TestCandlesCursorPagination(t *testing.T) {
	defer DeleteEverything()

	candleStore := sqlstore.NewCandles(context.Background(), connectionSource, candlesv2.NewDefaultConfig().CandleStore)
	tradeStore := sqlstore.NewTrades(connectionSource)

	startTime := time.Unix(StartTime, 0)
	insertCandlesTestData(t, tradeStore, startTime, totalBlocks, tradesPerBlock, startPrice, priceIncrement, size, blockIntervalDur)

	_, candleId, err := candleStore.GetCandleIdForIntervalAndMarket(context.Background(), "1 Minute", testMarket)
	if err != nil {
		t.Fatalf("getting existing candleDescriptor id:%s", err)
	}

	pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil)
	// retrieve all candles without pagination to use for test validation
	allCandles, _, err := candleStore.GetCandleDataForTimeSpan(context.Background(), candleId, &startTime,
		nil, pagination)
	if err != nil {
		t.Fatalf("failed to get candles:%s", err)
	}

	require.Equal(t, 167, len(allCandles))

	t.Run("should return the first candles when first is provided with no after", func(t *testing.T) {
		first := int32(10)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil)
		require.NoError(t, err)
		candles, pageInfo, err := candleStore.GetCandleDataForTimeSpan(context.Background(), candleId, &startTime, nil, pagination)
		require.NoError(t, err)
		require.Equal(t, 10, len(candles))
		assert.Equal(t, allCandles[0], candles[0])
		assert.Equal(t, allCandles[9], candles[9])
		assert.Equal(t, pageInfo.StartCursor, entities.NewCursor(allCandles[0].PeriodStart.Format(time.RFC3339Nano)).Encode())
		assert.Equal(t, pageInfo.EndCursor, entities.NewCursor(allCandles[9].PeriodStart.Format(time.RFC3339Nano)).Encode())
		assert.False(t, pageInfo.HasPreviousPage)
		assert.True(t, pageInfo.HasNextPage)
	})

	t.Run("should return the last page of candles when last is provided with no before", func(t *testing.T) {
		last := int32(10)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil)
		require.NoError(t, err)
		candles, pageInfo, err := candleStore.GetCandleDataForTimeSpan(context.Background(), candleId, &startTime, nil, pagination)
		require.NoError(t, err)
		require.Equal(t, 10, len(candles))
		assert.Equal(t, allCandles[157], candles[0])
		assert.Equal(t, allCandles[166], candles[9])
		assert.Equal(t, pageInfo.StartCursor, entities.NewCursor(allCandles[157].PeriodStart.Format(time.RFC3339Nano)).Encode())
		assert.Equal(t, pageInfo.EndCursor, entities.NewCursor(allCandles[166].PeriodStart.Format(time.RFC3339Nano)).Encode())
		assert.True(t, pageInfo.HasPreviousPage)
		assert.False(t, pageInfo.HasNextPage)
	})

	t.Run("should return the requested page of candles when first and after are provided", func(t *testing.T) {
		first := int32(10)
		after := entities.NewCursor(allCandles[99].PeriodStart.Format(time.RFC3339Nano)).Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil)
		require.NoError(t, err)
		candles, pageInfo, err := candleStore.GetCandleDataForTimeSpan(context.Background(), candleId, &startTime, nil, pagination)
		require.NoError(t, err)
		require.Equal(t, 10, len(candles))
		assert.Equal(t, allCandles[100], candles[0])
		assert.Equal(t, allCandles[109], candles[9])
		assert.Equal(t, pageInfo.StartCursor, entities.NewCursor(allCandles[100].PeriodStart.Format(time.RFC3339Nano)).Encode())
		assert.Equal(t, pageInfo.EndCursor, entities.NewCursor(allCandles[109].PeriodStart.Format(time.RFC3339Nano)).Encode())
		assert.True(t, pageInfo.HasPreviousPage)
		assert.True(t, pageInfo.HasNextPage)
	})

	t.Run("Should return the requested page of candles when last and before are provided", func(t *testing.T) {
		last := int32(10)
		before := entities.NewCursor(allCandles[100].PeriodStart.Format(time.RFC3339Nano)).Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before)
		require.NoError(t, err)
		candles, pageInfo, err := candleStore.GetCandleDataForTimeSpan(context.Background(), candleId, &startTime, nil, pagination)
		require.NoError(t, err)
		require.Equal(t, 10, len(candles))
		assert.Equal(t, allCandles[90], candles[0])
		assert.Equal(t, allCandles[99], candles[9])
		assert.Equal(t, pageInfo.StartCursor, entities.NewCursor(allCandles[90].PeriodStart.Format(time.RFC3339Nano)).Encode())
		assert.Equal(t, pageInfo.EndCursor, entities.NewCursor(allCandles[99].PeriodStart.Format(time.RFC3339Nano)).Encode())
		assert.True(t, pageInfo.HasPreviousPage)
		assert.True(t, pageInfo.HasNextPage)

	})
}
