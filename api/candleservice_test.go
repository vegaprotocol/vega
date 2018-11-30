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
//
//func TestCandleService_Generate(t *testing.T) {
//	testMarket := "BTC/DEC18"
//
//	var ctx = context.Background()
//	var candleService = NewCandleService()
//
//	FlushCandleStore()
//	candleStore := datastore.NewCandleStore(candleStoreDir)
//	defer candleStore.Close()
//
//	config := core.GetConfig()
//	vega := core.New(config, nil, nil, candleStore)
//	vega.InitialiseMarkets()
//
//	candleService.Init(vega, candleStore)
//
//	// t0 = 2018-11-13T11:01:14Z
//	t0 := uint64(1542106874000000000)
//	//t0Seconds := int64(1542106874)
//	//t0NanoSeconds := int64(000000000)
//	//t0stamp := time.Unix(t0Seconds, t0NanoSeconds)
//
//	var trades = []*msg.Trade{
//		{Id: "1", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0},
//		{Id: "2", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(20 * time.Second)},
//
//		{Id: "3", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute)},
//		{Id: "4", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute + 20 * time.Second)},
//	}
//
//	for idx := range trades {
//		candleService.AddTradeToBuffer(trades[idx])
//	}
//
//	candleService.Generate(ctx, testMarket)
//
//	// test for 1 minute intervals
//	candles, err := candleService.GetCandles(ctx, testMarket, t0, msg.Interval_I1M)
//	assert.Nil(t, err)
//
//	fmt.Printf("Candles fetched for t0 and 1m: %+v\n", candles)
//
//	assert.Equal(t, 2, len(candles))
//	fmt.Printf("%s", time.Unix(1542106860,000000000).Format(time.RFC3339))
//	assert.Equal(t, uint64(1542106860000000000), candles[0].Timestamp)
//	assert.Equal(t, uint64(100), candles[0].High)
//	assert.Equal(t, uint64(100), candles[0].Low)
//	assert.Equal(t, uint64(100), candles[0].Open)
//	assert.Equal(t, uint64(100), candles[0].Close)
//	assert.Equal(t, uint64(200), candles[0].Volume)
//
//	assert.Equal(t, uint64(1542106920000000000), candles[1].Timestamp)
//	assert.Equal(t, uint64(100), candles[1].High)
//	assert.Equal(t, uint64(100), candles[1].Low)
//	assert.Equal(t, uint64(100), candles[1].Open)
//	assert.Equal(t, uint64(100), candles[1].Close)
//	assert.Equal(t, uint64(200), candles[1].Volume)
//}

func TestCandleService_ObserveCandles(t *testing.T) {
	MarketBTC := "BTC/DEC18"
	MarketETH := "ETH/APR19"
	var ctx = context.Background()
	var candleService = NewCandleService()

	FlushCandleStore()
	candleStore := datastore.NewCandleStore(candleStoreDir)
	defer candleStore.Close()

	config := core.GetConfig()
	vega := core.New(config, nil, nil, candleStore)
	vega.InitialiseMarkets()

	candleService.Init(vega, candleStore)

	interval1m := msg.Interval_I1M
	interval5m := msg.Interval_I5M
	interval15m := msg.Interval_I15M
	interval1h := msg.Interval_I1H
	interval6h := msg.Interval_I6H
	interval1d := msg.Interval_I1D

	// -------- BTC MARKET SUBSCRIPTIONS -----

	candlesSubscription1m_BTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval1m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1m_BTC))
	assert.Equal(t, uint64(1), ref)

	candlesSubscription5m_BTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval5m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription5m_BTC))
	assert.Equal(t, uint64(2), ref)

	candlesSubscription15m_BTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval15m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription15m_BTC))
	assert.Equal(t, uint64(3), ref)

	candlesSubscription1h_BTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval1h)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1h_BTC))
	assert.Equal(t, uint64(4), ref)

	candlesSubscription6h_BTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval6h)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription6h_BTC))
	assert.Equal(t, uint64(5), ref)

	candlesSubscription1d_BTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval1d)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1d_BTC))
	assert.Equal(t, uint64(6), ref)


	// -------- ETH MARKET SUBSCRIPTIONS -----

	candlesSubscription1m_ETH, ref := candleService.ObserveCandles(ctx, &MarketETH, &interval1m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1m_ETH))
	assert.Equal(t, uint64(7), ref)

	candlesSubscription5m_ETH, ref := candleService.ObserveCandles(ctx, &MarketETH, &interval5m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription5m_ETH))
	assert.Equal(t, uint64(8), ref)

	candlesSubscription15m_ETH, ref := candleService.ObserveCandles(ctx, &MarketETH, &interval15m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription15m_ETH))
	assert.Equal(t, uint64(9), ref)

	candlesSubscription1h_ETH, ref := candleService.ObserveCandles(ctx, &MarketETH, &interval1h)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1h_ETH))
	assert.Equal(t, uint64(10), ref)

	candlesSubscription6h_ETH, ref := candleService.ObserveCandles(ctx, &MarketETH, &interval6h)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription6h_ETH))
	assert.Equal(t, uint64(11), ref)

	candlesSubscription1d_ETH, ref := candleService.ObserveCandles(ctx, &MarketETH, &interval1d)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1d_ETH))
	assert.Equal(t, uint64(12), ref)


	// t0 = 2018-11-13T11:01:14Z
	t0 := uint64(1542106874000000000)

	go func() {
		for {
			select {
			case candle := <- candlesSubscription1m_BTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0 - uint64(14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, msg.Interval_I1M, candle.Interval)

			case candle := <- candlesSubscription5m_BTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, msg.Interval_I5M, candle.Interval)

			case candle := <- candlesSubscription15m_BTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, msg.Interval_I15M, candle.Interval)

			case candle := <- candlesSubscription1h_BTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, msg.Interval_I1H, candle.Interval)

			case candle := <- candlesSubscription6h_BTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0 - uint64(5 * time.Hour + time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, msg.Interval_I6H, candle.Interval)

			case candle := <- candlesSubscription1d_BTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0 - uint64(11 * time.Hour + time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, msg.Interval_I1D, candle.Interval)

			case candle := <- candlesSubscription1m_ETH:
				fmt.Printf("RECEIVED CANDLE ETH %+v\n", candle)
				assert.Equal(t, t0 - uint64(14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, msg.Interval_I1M, candle.Interval)

			case candle := <- candlesSubscription5m_ETH:
				fmt.Printf("RECEIVED CANDLE ETH %+v\n", candle)
				assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, msg.Interval_I5M, candle.Interval)

			case candle := <- candlesSubscription15m_ETH:
				fmt.Printf("RECEIVED CANDLE ETH %+v\n", candle)
				assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, msg.Interval_I15M, candle.Interval)

			case candle := <- candlesSubscription1h_ETH:
				fmt.Printf("RECEIVED CANDLE ETH %+v\n", candle)
				assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, msg.Interval_I1H, candle.Interval)

			case candle := <- candlesSubscription6h_ETH:
				fmt.Printf("RECEIVED CANDLE ETH %+v\n", candle)
				assert.Equal(t, t0 - uint64(5 * time.Hour + time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, msg.Interval_I6H, candle.Interval)

			case candle := <- candlesSubscription1d_ETH:
				fmt.Printf("RECEIVED CANDLE ETH %+v\n", candle)
				assert.Equal(t, t0 - uint64(11 * time.Hour + time.Minute + 14 * time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, msg.Interval_I1D, candle.Interval)

			}
		}
	}()

	var trades = []*msg.Trade{
		{Id: "1", Market: MarketBTC, Price: uint64(100), Size: uint64(100), Timestamp: t0},
		{Id: "2", Market: MarketETH, Price: uint64(100), Size: uint64(100), Timestamp: t0},
		{Id: "3", Market: MarketBTC, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(20 * time.Second)},
		{Id: "4", Market: MarketETH, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(20 * time.Second)},
	}
	//
	candleStore.StartNewBuffer(MarketBTC, t0)
	candleStore.StartNewBuffer(MarketETH, t0)
	for idx := range trades {
		candleStore.AddTradeToBuffer(trades[idx].Market, *trades[idx])
	}
	candleStore.GenerateCandlesFromBuffer(MarketBTC)
	candleStore.GenerateCandlesFromBuffer(MarketETH)

	time.Sleep(1*time.Second)
	fmt.Printf("End of test\n")
}

func isSubscriptionEmpty(transport <-chan msg.Candle) bool {
	select {
	case  <- transport:
		return false
	default:
		return true
	}
}