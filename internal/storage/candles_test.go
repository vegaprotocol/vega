package storage_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/buffer"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	storcfg "code.vegaprotocol.io/vega/internal/storage/config"
	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func NewTestCandlesConfig(t *testing.T) storcfg.CandlesConfig {
	return storcfg.NewDefaultCandlesConfig(tempDir(t, "testcandlestore"))
}

func runCandleStoreTest(t *testing.T, test func(t *testing.T, candleStore *storage.Candle)) {
	log := logging.NewTestLogger()
	cfg := NewTestCandlesConfig(t)
	cs, err := storage.NewCandles(log, cfg)
	require.NoError(t, err)
	defer os.RemoveAll(cfg.Storage.Path)
	defer cs.Close()

	test(t, cs)
}

func TestStorage_GenerateCandles(t *testing.T) {
	runCandleStoreTest(t, func(t *testing.T, candleStore *storage.Candle) {
		ctx := context.Background()

		// t0 = 2018-11-13T11:01:14Z
		t0 := vegatime.UnixNano(1542106874000000000)
		t.Log(fmt.Sprintf("t0 = %s", vegatime.Format(t0)))

		var trades = []*types.Trade{
			{Id: "1", MarketID: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0.UnixNano()},
			{Id: "2", MarketID: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0.Add(20 * time.Second).UnixNano()},
			{Id: "3", MarketID: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0.Add(1 * time.Minute).UnixNano()},
			{Id: "4", MarketID: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0.Add(1*time.Minute + 20*time.Second).UnixNano()},
		}

		// create+start a new buffer
		buf := buffer.NewCandle(testMarket, candleStore, t0)
		// assert.Nil(t, err)

		for idx := range trades {
			err := buf.AddTrade(*trades[idx])
			assert.Nil(t, err)
		}

		// start a new buffer, to get the previous one
		currentVegaTime := t0.Add(2 * time.Minute)
		previousBuf, err := buf.Start(currentVegaTime)

		err = candleStore.GenerateCandlesFromBuffer(testMarket, previousBuf)
		assert.Nil(t, err)

		candles, err := candleStore.GetCandles(ctx, testMarket, t0, types.Interval_I1M)
		t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))
		assert.Nil(t, err)

		assert.Equal(t, 2, len(candles))

		t.Log(fmt.Sprintf("%s", vegatime.Format(time.Unix(1542106860, 000000000))))
		assert.Equal(t, int64(1542106860000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(200), candles[0].Volume)

		assert.Equal(t, int64(1542106920000000000), candles[1].Timestamp)
		assert.Equal(t, uint64(100), candles[1].High)
		assert.Equal(t, uint64(100), candles[1].Low)
		assert.Equal(t, uint64(100), candles[1].Open)
		assert.Equal(t, uint64(100), candles[1].Close)
		assert.Equal(t, uint64(200), candles[1].Volume)

		candles, err = candleStore.GetCandles(ctx, testMarket, t0.Add(1*time.Minute), types.Interval_I1M)
		assert.Nil(t, err)
		t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

		assert.Equal(t, 1, len(candles))
		assert.Equal(t, int64(1542106920000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(200), candles[0].Volume)

		candles, err = candleStore.GetCandles(ctx, testMarket, t0.Add(1*time.Minute), types.Interval_I5M)
		assert.Nil(t, err)
		t.Log(fmt.Sprintf("Candles fetched for t0 and 5m: %+v", candles))

		assert.Equal(t, 1, len(candles))
		assert.Equal(t, int64(1542106800000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)

		//------------------- generate empty candles-------------------------//
		// currentVegaTime := t0.Add(2 * time.Minute)
		// err = candleStore.StartNewBuffer(testMarket, currentVegaTime)
		assert.Nil(t, err)
		// we use the buffer started previously when stopping the previous test
		currentVegaTime = t0.Add(17 * time.Minute)
		previousBuf, _ = buf.Start(currentVegaTime)
		err = candleStore.GenerateCandlesFromBuffer(testMarket, previousBuf)
		assert.Nil(t, err)

		candles, err = candleStore.GetCandles(ctx, testMarket, t0, types.Interval_I1M)
		assert.Nil(t, err)
		t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

		fmt.Println(" --- candles ", fmt.Sprintf("%+v", candles))

		assert.Equal(t, 3, len(candles))
		assert.Equal(t, int64(1542106860000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(200), candles[0].Volume)

		assert.Equal(t, int64(1542106920000000000), candles[1].Timestamp)
		assert.Equal(t, uint64(100), candles[1].High)
		assert.Equal(t, uint64(100), candles[1].Low)
		assert.Equal(t, uint64(100), candles[1].Open)
		assert.Equal(t, uint64(100), candles[1].Close)
		assert.Equal(t, uint64(200), candles[1].Volume)

		assert.Equal(t, int64(1542106980000000000), candles[2].Timestamp)
		assert.Equal(t, uint64(100), candles[2].High)
		assert.Equal(t, uint64(100), candles[2].Low)
		assert.Equal(t, uint64(100), candles[2].Open)
		assert.Equal(t, uint64(100), candles[2].Close)
		assert.Equal(t, uint64(0), candles[2].Volume)

		candles, err = candleStore.GetCandles(ctx, testMarket, t0, types.Interval_I5M)
		assert.Nil(t, err)
		t.Log(fmt.Sprintf("Candles fetched for t0 and 5m: %+v", candles))

		assert.Equal(t, 1, len(candles))
		assert.Equal(t, int64(1542106800000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)

		candles, err = candleStore.GetCandles(ctx, testMarket, t0.Add(2*time.Minute), types.Interval_I15M)
		assert.Nil(t, err)
		t.Log(fmt.Sprintf("Candles fetched for t0 and 15m: %+v", candles))

		assert.Equal(t, 1, len(candles))
		assert.Equal(t, int64(1542106800000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)

		candles, err = candleStore.GetCandles(ctx, testMarket, t0.Add(17*time.Minute), types.Interval_I15M)
		assert.Nil(t, err)
		t.Log(fmt.Sprintf("Candles fetched for t0 and 15m: %+v", candles))

		assert.Equal(t, 0, len(candles))

		currentVegaTime = t0.Add(20 * time.Minute)
		previousBuf, err = buf.Start(currentVegaTime)
		assert.Nil(t, err)
		err = candleStore.GenerateCandlesFromBuffer(testMarket, previousBuf)
		assert.Nil(t, err)

		candles, err = candleStore.GetCandles(ctx, testMarket, t0.Add(17*time.Minute), types.Interval_I15M)
		assert.Nil(t, err)
		t.Log(fmt.Sprintf("Candles fetched for t0 and 15m: %+v", candles))

		assert.Equal(t, 1, len(candles))
		assert.Equal(t, int64(1542107700000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(0), candles[0].Volume)
	})
}

func TestStorage_GetMapOfIntervalsToTimestamps(t *testing.T) {
	timestamp, _ := vegatime.Parse("2018-11-13T11:01:14Z")
	t0 := timestamp
	timestamps := buffer.GetMapOfIntervalsToRoundedTimestamps(timestamp)
	assert.Equal(t, t0.Add(-14*time.Second), timestamps[types.Interval_I1M])
	assert.Equal(t, t0.Add(-(time.Minute + 14*time.Second)), timestamps[types.Interval_I5M])
	assert.Equal(t, t0.Add(-(time.Minute + 14*time.Second)), timestamps[types.Interval_I15M])
	assert.Equal(t, t0.Add(-(time.Minute + 14*time.Second)), timestamps[types.Interval_I1H])
	assert.Equal(t, t0.Add(-(5*time.Hour + time.Minute + 14*time.Second)), timestamps[types.Interval_I6H])
	assert.Equal(t, t0.Add(-(11*time.Hour + time.Minute + 14*time.Second)), timestamps[types.Interval_I1D])
}

func TestStorage_SubscribeUnsubscribeCandles(t *testing.T) {
	runCandleStoreTest(t, func(t *testing.T, candleStore *storage.Candle) {
		internalTransport1 := &storage.InternalTransport{
			Market:    testMarket,
			Interval:  types.Interval_I1M,
			Transport: make(chan *types.Candle)}
		ref := candleStore.Subscribe(internalTransport1)
		assert.Equal(t, uint64(1), ref)

		internalTransport2 := &storage.InternalTransport{
			Market:    testMarket,
			Interval:  types.Interval_I1M,
			Transport: make(chan *types.Candle)}
		ref = candleStore.Subscribe(internalTransport2)
		assert.Equal(t, uint64(2), ref)

		err := candleStore.Unsubscribe(1)
		assert.Nil(t, err)

		err = candleStore.Unsubscribe(1)
		assert.Equal(t, "Candle store subscriber does not exist with id: 1", err.Error())

		err = candleStore.Unsubscribe(2)
		assert.Nil(t, err)

		err = candleStore.Unsubscribe(2)
		assert.Nil(t, err)
	})
}

func TestStorage_PreviousCandleDerivedValues(t *testing.T) {
	runCandleStoreTest(t, func(t *testing.T, candleStore *storage.Candle) {
		ctx := context.Background()
		// t0 = 2018-11-13T11:00:00Z
		t0 := vegatime.UnixNano(1542106800000000000)

		var trades1 = []*types.Trade{
			{Id: "1", MarketID: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0.UnixNano()},
			{Id: "2", MarketID: testMarket, Price: uint64(99), Size: uint64(100), Timestamp: t0.Add(10 * time.Second).UnixNano()},
			{Id: "3", MarketID: testMarket, Price: uint64(108), Size: uint64(100), Timestamp: t0.Add(20 * time.Second).UnixNano()},
			{Id: "4", MarketID: testMarket, Price: uint64(105), Size: uint64(100), Timestamp: t0.Add(30 * time.Second).UnixNano()},
			{Id: "5", MarketID: testMarket, Price: uint64(110), Size: uint64(100), Timestamp: t0.Add(1 * time.Minute).UnixNano()},
			{Id: "6", MarketID: testMarket, Price: uint64(112), Size: uint64(100), Timestamp: t0.Add(1*time.Minute + 10*time.Second).UnixNano()},
			{Id: "7", MarketID: testMarket, Price: uint64(113), Size: uint64(100), Timestamp: t0.Add(1*time.Minute + 20*time.Second).UnixNano()},
			{Id: "8", MarketID: testMarket, Price: uint64(109), Size: uint64(100), Timestamp: t0.Add(1*time.Minute + 30*time.Second).UnixNano()},
		}

		buf := buffer.NewCandle(testMarket, candleStore, t0)
		for idx := range trades1 {
			err := buf.AddTrade(*trades1[idx])
			assert.Nil(t, err)
		}
		previousBuf, _ := buf.Start(t0.Add(2 * time.Minute))
		err := candleStore.GenerateCandlesFromBuffer(testMarket, previousBuf)
		assert.Nil(t, err)

		candles, err := candleStore.GetCandles(ctx, testMarket, t0, types.Interval_I1M)
		assert.Nil(t, err)

		t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

		assert.Equal(t, 2, len(candles))

		t.Log(fmt.Sprintf("%s", vegatime.Format(time.Unix(1542106860, 000000000))))

		assert.Equal(t, t0.UnixNano(), candles[0].Timestamp)
		assert.Equal(t, uint64(108), candles[0].High)
		assert.Equal(t, uint64(99), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(105), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)

		assert.Equal(t, t0.Add(1*time.Minute).UnixNano(), candles[1].Timestamp)
		assert.Equal(t, uint64(113), candles[1].High)
		assert.Equal(t, uint64(109), candles[1].Low)
		assert.Equal(t, uint64(110), candles[1].Open)
		assert.Equal(t, uint64(109), candles[1].Close)
		assert.Equal(t, uint64(400), candles[1].Volume)

		candles, err = candleStore.GetCandles(ctx, testMarket, t0.Add(1*time.Minute), types.Interval_I1M)
		assert.Nil(t, err)

		t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

		assert.Equal(t, 1, len(candles))
		assert.Equal(t, t0.Add(1*time.Minute).UnixNano(), candles[0].Timestamp)
		assert.Equal(t, uint64(113), candles[0].High)
		assert.Equal(t, uint64(109), candles[0].Low)
		assert.Equal(t, uint64(110), candles[0].Open)
		assert.Equal(t, uint64(109), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)

		candles, err = candleStore.GetCandles(ctx, testMarket, t0, types.Interval_I5M)
		assert.Nil(t, err)

		t.Log(fmt.Sprintf("Candles fetched for t0 and 5m: %+v", candles))

		assert.Equal(t, 1, len(candles))
		assert.Equal(t, t0.UnixNano(), candles[0].Timestamp)
		assert.Equal(t, uint64(113), candles[0].High)
		assert.Equal(t, uint64(99), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(109), candles[0].Close)
		assert.Equal(t, uint64(800), candles[0].Volume)

		var trades2 = []*types.Trade{
			{Id: "9", MarketID: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0.Add(2*time.Minute + 10*time.Second).UnixNano()},
			{Id: "10", MarketID: testMarket, Price: uint64(99), Size: uint64(100), Timestamp: t0.Add(2*time.Minute + 20*time.Second).UnixNano()},
			{Id: "11", MarketID: testMarket, Price: uint64(108), Size: uint64(100), Timestamp: t0.Add(2*time.Minute + 30*time.Second).UnixNano()},
			{Id: "12", MarketID: testMarket, Price: uint64(105), Size: uint64(100), Timestamp: t0.Add(2*time.Minute + 40*time.Second).UnixNano()},
			{Id: "13", MarketID: testMarket, Price: uint64(110), Size: uint64(100), Timestamp: t0.Add(3*time.Minute + 10*time.Second).UnixNano()},
			{Id: "14", MarketID: testMarket, Price: uint64(112), Size: uint64(100), Timestamp: t0.Add(3*time.Minute + 20*time.Second).UnixNano()},
			{Id: "15", MarketID: testMarket, Price: uint64(113), Size: uint64(100), Timestamp: t0.Add(3*time.Minute + 30*time.Second).UnixNano()},
			{Id: "16", MarketID: testMarket, Price: uint64(109), Size: uint64(100), Timestamp: t0.Add(3*time.Minute + 40*time.Second).UnixNano()},
		}

		assert.Nil(t, err)
		for idx := range trades2 {
			err := buf.AddTrade(*trades2[idx])
			assert.Nil(t, err)
		}
		previousBuf, _ = buf.Start(t0.Add(4 * time.Minute))
		err = candleStore.GenerateCandlesFromBuffer(testMarket, previousBuf)
		assert.Nil(t, err)

		candles, err = candleStore.GetCandles(ctx, testMarket, t0, types.Interval_I1M)
		assert.Nil(t, err)

		t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

		assert.Equal(t, t0.UnixNano(), candles[0].Timestamp)
		assert.Equal(t, uint64(108), candles[0].High)
		assert.Equal(t, uint64(99), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(105), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)

		assert.Equal(t, t0.Add(1*time.Minute).UnixNano(), candles[1].Timestamp)
		assert.Equal(t, uint64(113), candles[1].High)
		assert.Equal(t, uint64(109), candles[1].Low)
		assert.Equal(t, uint64(110), candles[1].Open)
		assert.Equal(t, uint64(109), candles[1].Close)
		assert.Equal(t, uint64(400), candles[1].Volume)

		assert.Equal(t, t0.Add(2*time.Minute).UnixNano(), candles[2].Timestamp)
		assert.Equal(t, uint64(108), candles[2].High)
		assert.Equal(t, uint64(99), candles[2].Low)
		assert.Equal(t, uint64(100), candles[2].Open)
		assert.Equal(t, uint64(105), candles[2].Close)
		assert.Equal(t, uint64(400), candles[2].Volume)

		assert.Equal(t, t0.Add(3*time.Minute).UnixNano(), candles[3].Timestamp)
		assert.Equal(t, uint64(113), candles[3].High)
		assert.Equal(t, uint64(109), candles[3].Low)
		assert.Equal(t, uint64(110), candles[3].Open)
		assert.Equal(t, uint64(109), candles[3].Close)
		assert.Equal(t, uint64(400), candles[3].Volume)

		var trades3 = []*types.Trade{
			{Id: "17", MarketID: testMarket, Price: uint64(95), Size: uint64(100), Timestamp: t0.Add(4*time.Minute + 10*time.Second).UnixNano()},
			{Id: "18", MarketID: testMarket, Price: uint64(80), Size: uint64(100), Timestamp: t0.Add(4*time.Minute + 20*time.Second).UnixNano()},
			{Id: "19", MarketID: testMarket, Price: uint64(120), Size: uint64(100), Timestamp: t0.Add(4*time.Minute + 30*time.Second).UnixNano()},
			{Id: "20", MarketID: testMarket, Price: uint64(105), Size: uint64(100), Timestamp: t0.Add(4*time.Minute + 40*time.Second).UnixNano()},
			{Id: "21", MarketID: testMarket, Price: uint64(103), Size: uint64(100), Timestamp: t0.Add(5*time.Minute + 10*time.Second).UnixNano()},
			{Id: "22", MarketID: testMarket, Price: uint64(101), Size: uint64(100), Timestamp: t0.Add(5*time.Minute + 20*time.Second).UnixNano()},
			{Id: "23", MarketID: testMarket, Price: uint64(101), Size: uint64(100), Timestamp: t0.Add(5*time.Minute + 30*time.Second).UnixNano()},
			{Id: "24", MarketID: testMarket, Price: uint64(101), Size: uint64(100), Timestamp: t0.Add(5*time.Minute + 40*time.Second).UnixNano()},
		}

		assert.Nil(t, err)
		for idx := range trades3 {
			err := buf.AddTrade(*trades3[idx])
			assert.Nil(t, err)
		}
		previousBuf, _ = buf.Start(t0.Add(10 * time.Minute))
		err = candleStore.GenerateCandlesFromBuffer(testMarket, previousBuf)
		assert.Nil(t, err)

		candles, err = candleStore.GetCandles(ctx, testMarket, t0, types.Interval_I1M)
		assert.Nil(t, err)

		t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

		assert.Equal(t, t0.UnixNano(), candles[0].Timestamp)
		assert.Equal(t, uint64(108), candles[0].High)
		assert.Equal(t, uint64(99), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(105), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)

		assert.Equal(t, t0.Add(1*time.Minute).UnixNano(), candles[1].Timestamp)
		assert.Equal(t, uint64(113), candles[1].High)
		assert.Equal(t, uint64(109), candles[1].Low)
		assert.Equal(t, uint64(110), candles[1].Open)
		assert.Equal(t, uint64(109), candles[1].Close)
		assert.Equal(t, uint64(400), candles[1].Volume)

		assert.Equal(t, t0.Add(2*time.Minute).UnixNano(), candles[2].Timestamp)
		assert.Equal(t, uint64(108), candles[2].High)
		assert.Equal(t, uint64(99), candles[2].Low)
		assert.Equal(t, uint64(100), candles[2].Open)
		assert.Equal(t, uint64(105), candles[2].Close)
		assert.Equal(t, uint64(400), candles[2].Volume)

		assert.Equal(t, t0.Add(3*time.Minute).UnixNano(), candles[3].Timestamp)
		assert.Equal(t, uint64(113), candles[3].High)
		assert.Equal(t, uint64(109), candles[3].Low)
		assert.Equal(t, uint64(110), candles[3].Open)
		assert.Equal(t, uint64(109), candles[3].Close)
		assert.Equal(t, uint64(400), candles[3].Volume)

		assert.Equal(t, t0.Add(4*time.Minute).UnixNano(), candles[4].Timestamp)
		assert.Equal(t, uint64(120), candles[4].High)
		assert.Equal(t, uint64(80), candles[4].Low)
		assert.Equal(t, uint64(95), candles[4].Open)
		assert.Equal(t, uint64(105), candles[4].Close)
		assert.Equal(t, uint64(400), candles[4].Volume)

		assert.Equal(t, t0.Add(5*time.Minute).UnixNano(), candles[5].Timestamp)
		assert.Equal(t, uint64(103), candles[5].High)
		assert.Equal(t, uint64(101), candles[5].Low)
		assert.Equal(t, uint64(103), candles[5].Open)
		assert.Equal(t, uint64(101), candles[5].Close)
		assert.Equal(t, uint64(400), candles[5].Volume)
	})
}
