package api

import (
	"context"
	"testing"
	"vega/msg"
	"time"
	"vega/datastore"
	"fmt"
	"os"
	"vega/core"
	"github.com/stretchr/testify/assert"
)

const candleStoreDir = "../tmp/candlestore-api"

func FlushCandleStore() {
	fmt.Printf("Flushing candle store\n")
	err := os.RemoveAll(candleStoreDir)
	if err != nil {
		fmt.Printf("UNABLE TO FLUSH DB: %s\n", err.Error())
	}
}

func TestCandleService_Generate(t *testing.T) {
	testMarket := "BTC/DEC18"

	var ctx = context.Background()
	var candleService = NewCandleService()

	FlushCandleStore()
	candleStore := datastore.NewCandleStore(candleStoreDir)
	defer candleStore.Close()

	config := core.GetConfig()
	vega := core.New(config, nil, nil, candleStore)
	vega.InitialiseMarkets()

	candleService.Init(vega, candleStore)

	// t0 = 2018-11-13T11:01:14Z
	t0 := uint64(1542106874000000000)
	t0Seconds := int64(1542106874)
	t0NanoSeconds := int64(000000000)
	t0stamp := time.Unix(t0Seconds, t0NanoSeconds)

	var trades = []*msg.Trade{
		{Id: "1", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0},
		{Id: "2", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(20 * time.Second)},

		{Id: "3", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute)},
		{Id: "4", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute + 20 * time.Second)},
	}

	for idx := range trades {
		candleService.AddTrade(trades[idx])
	}

	candleService.Generate(ctx, testMarket)

	// test for 1 minute intervals
	candles, err := candleService.GetCandles(ctx, testMarket, t0stamp, "1m")
	assert.Nil(t, err)

	fmt.Printf("Candles fetched for t0 and 1m: %+v\n", candles)

	assert.Equal(t, 2, len(candles))
	fmt.Printf("%s", time.Unix(1542106860,000000000).Format(time.RFC3339))
	assert.Equal(t, uint64(1542106860000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(200), candles[0].Volume)

	assert.Equal(t, uint64(1542106920000000000), candles[1].Timestamp)
	assert.Equal(t, uint64(100), candles[1].High)
	assert.Equal(t, uint64(100), candles[1].Low)
	assert.Equal(t, uint64(100), candles[1].Open)
	assert.Equal(t, uint64(100), candles[1].Close)
	assert.Equal(t, uint64(200), candles[1].Volume)
}

func TestCandleService_ObserveCandles(t *testing.T) {
	testMarket := "BTC/DEC18"
	var ctx = context.Background()
	var candleService = NewCandleService()

	FlushCandleStore()
	candleStore := datastore.NewCandleStore(candleStoreDir)
	defer candleStore.Close()

	config := core.GetConfig()
	vega := core.New(config, nil, nil, candleStore)
	vega.InitialiseMarkets()

	candleService.Init(vega, candleStore)

	interval1m := "1m"
	candlesSubscription1m, ref := candleService.ObserveCandles(ctx, &testMarket, &interval1m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1m))
	assert.Equal(t, uint64(1), ref)

	interval5m := "5m"
	candlesSubscription5m, ref := candleService.ObserveCandles(ctx, &testMarket, &interval5m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription5m))
	assert.Equal(t, uint64(2), ref)

	interval15m := "15m"
	candlesSubscription15m, ref := candleService.ObserveCandles(ctx, &testMarket, &interval15m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription15m))
	assert.Equal(t, uint64(3), ref)

	interval1h := "1h"
	candlesSubscription1h, ref := candleService.ObserveCandles(ctx, &testMarket, &interval1h)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1h))
	assert.Equal(t, uint64(4), ref)

	interval6h := "6h"
	candlesSubscription6h, ref := candleService.ObserveCandles(ctx, &testMarket, &interval6h)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription6h))
	assert.Equal(t, uint64(5), ref)

	interval1d := "1d"
	candlesSubscription1d, ref := candleService.ObserveCandles(ctx, &testMarket, &interval1d)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1d))
	assert.Equal(t, uint64(6), ref)

	// t0 = 2018-11-13T11:01:14Z
	t0 := uint64(1542106874000000000)

	go func() {
		for {
			select {
			case candle := <- candlesSubscription1m:
				fmt.Printf("RECEIVED CANDLE %+v\n", candle)
				assert.Equal(t, t0 - uint64(14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, "1m", candle.Interval)

			case candle := <- candlesSubscription5m:
				fmt.Printf("RECEIVED CANDLE %+v\n", candle)
				assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, "5m", candle.Interval)

			case candle := <- candlesSubscription15m:
				fmt.Printf("RECEIVED CANDLE %+v\n", candle)
				assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, "15m", candle.Interval)

			case candle := <- candlesSubscription1h:
				fmt.Printf("RECEIVED CANDLE %+v\n", candle)
				assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, "1h", candle.Interval)

			case candle := <- candlesSubscription6h:
				fmt.Printf("RECEIVED CANDLE %+v\n", candle)
				assert.Equal(t, t0 - uint64(5 * time.Hour + time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, "6h", candle.Interval)

			case candle := <- candlesSubscription1d:
				fmt.Printf("RECEIVED CANDLE %+v\n", candle)
				assert.Equal(t, t0 - uint64(11 * time.Hour + time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, "1d", candle.Interval)
			}
		}
	}()

	var trades = []*msg.Trade{
		{Id: "1", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0},
		{Id: "2", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(20 * time.Second)},
	}

	for idx := range trades {
		candleService.AddTrade(trades[idx])
	}

	candleService.Generate(ctx, testMarket)
}

func isSubscriptionEmpty(transport <-chan msg.Candle) bool {
	select {
	case  <- transport:
		return false
	default:
		return true
	}
}