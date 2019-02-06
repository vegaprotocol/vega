package candles

import (
	"context"
	"testing"
	"vega/msg"
	"time"
	"fmt"
	"github.com/stretchr/testify/assert"
	"vega/internal/storage"
)

// storageConfig specifies that the badger files are kept in a different
// directory when the candle service tests run. This is useful as when
// all the unit tests are run for the project they can be run in parallel.
func storageConfig() *storage.Config {
	storeConfig := storage.NewTestConfig()
	storeConfig.CandleStoreDirPath = "../storage/tmp/candlestore-2m9d0"
	storeConfig.OrderStoreDirPath = "../storage/tmp/orderstore-2m9d0"
	storeConfig.TradeStoreDirPath = "../storage/tmp/tradestore-2m9d0"
	return storeConfig
}

func TestCandleService_ObserveCandles(t *testing.T) {
	MarketBTC := "BTC/DEC19"
	MarketETH := "ETH/APR19"
	var ctx = context.Background()

	storeConfig := storageConfig()
	storage.FlushStores(storeConfig)
	candleStore, err :=storage.NewCandleStore(storeConfig)
	assert.Nil(t, err)
	defer candleStore.Close()

	var candleService = NewCandleService(candleStore)

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

func TestSubscriptionUpdates_MinMax(t *testing.T) {
	MarketBTC := "BTC/DEC19"
	var ctx= context.Background()
	storeConfig := storageConfig()
	storage.FlushStores(storeConfig)
	candleStore, err :=storage.NewCandleStore(storeConfig)
	assert.Nil(t, err)
	defer candleStore.Close()

	var candleService = NewCandleService(candleStore)

	interval5m := msg.Interval_I5M

	candlesSubscription5m_BTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval5m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription5m_BTC))
	assert.Equal(t, uint64(1), ref)

	// t0 = 2018-11-13T11:00:00Z
	t0 := uint64(1542106800000000000)

	// first update
	var trades1 = []*msg.Trade{
		{Id: "1", Market: MarketBTC, Price: uint64(100), Size: uint64(100), Timestamp: t0},
		{Id: "2", Market: MarketBTC, Price: uint64(99), Size: uint64(100), Timestamp: t0 + uint64(10 * time.Second)},
		{Id: "3", Market: MarketBTC, Price: uint64(108), Size: uint64(100), Timestamp: t0 + uint64(20 * time.Second)},
		{Id: "4", Market: MarketBTC, Price: uint64(105), Size: uint64(100), Timestamp: t0 + uint64(30 * time.Second)},
	}

	// second update
	var trades2 = []*msg.Trade{
		{Id: "5", Market: MarketBTC, Price: uint64(110), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute)},
		{Id: "6", Market: MarketBTC, Price: uint64(112), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute + 10 * time.Second)},
		{Id: "7", Market: MarketBTC, Price: uint64(113), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute + 20 * time.Second)},
		{Id: "8", Market: MarketBTC, Price: uint64(109), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute + 30 * time.Second)},
	}

	// third update
	var trades3 = []*msg.Trade{
		{Id: "9", Market: MarketBTC, Price: uint64(110), Size: uint64(100), Timestamp: t0 + uint64(2 * time.Minute)},
		{Id: "10", Market: MarketBTC, Price: uint64(115), Size: uint64(100), Timestamp: t0 + uint64(2 * time.Minute + 10 * time.Second)},
		{Id: "11", Market: MarketBTC, Price: uint64(90), Size: uint64(100), Timestamp: t0 + uint64(2 * time.Minute + 20 * time.Second)},
		{Id: "12", Market: MarketBTC, Price: uint64(95), Size: uint64(100), Timestamp: t0 + uint64(2 * time.Minute + 30 * time.Second)},
	}

	listenToCandles := func(u1, u2, u3 *bool) {
		for {
			select {
			case candle := <-candlesSubscription5m_BTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0, candle.Timestamp)
				assert.Equal(t, msg.Interval_I5M, candle.Interval)

				switch candle.Volume {
				case uint64(400):
					fmt.Printf("RECEIVED CANDLE UPDATE 1\n")
					assert.Equal(t, uint64(100), candle.Open)
					assert.Equal(t, uint64(99), candle.Low)
					assert.Equal(t, uint64(108), candle.High)
					assert.Equal(t, uint64(105), candle.Close)
					*u1 = true

				case uint64(800):
					fmt.Printf("RECEIVED CANDLE UPDATE 2\n")
					assert.Equal(t, uint64(100), candle.Open)
					assert.Equal(t, uint64(99), candle.Low)
					assert.Equal(t, uint64(113), candle.High)
					assert.Equal(t, uint64(109), candle.Close)
					*u2 = true

				case uint64(1200):
					fmt.Printf("RECEIVED CANDLE UPDATE 3\n")
					assert.Equal(t, uint64(100), candle.Open)
					assert.Equal(t, uint64(90), candle.Low)
					assert.Equal(t, uint64(115), candle.High)
					assert.Equal(t, uint64(95), candle.Close)
					*u3 = true
				}
			}
		}
	}

	var (
		u1, u2, u3 = false, false, false
	)
	go listenToCandles(&u1, &u2, &u3)

	// first update
	candleStore.StartNewBuffer(MarketBTC, t0)
	for idx := range trades1 {
		candleStore.AddTradeToBuffer(trades1[idx].Market, *trades1[idx])
	}
	candleStore.GenerateCandlesFromBuffer(MarketBTC)

	// second update
	candleStore.StartNewBuffer(MarketBTC, t0 + uint64(1 * time.Minute))
	for idx := range trades2 {
		candleStore.AddTradeToBuffer(trades2[idx].Market, *trades2[idx])
	}
	candleStore.GenerateCandlesFromBuffer(MarketBTC)

	// third update
	candleStore.StartNewBuffer(MarketBTC, t0 + uint64(1 * time.Minute))
	for idx := range trades3 {
		candleStore.AddTradeToBuffer(trades3[idx].Market, *trades3[idx])
	}
	candleStore.GenerateCandlesFromBuffer(MarketBTC)

	time.Sleep(3 * time.Second)
	assert.True(t, u1)
	assert.True(t, u2)
	assert.True(t, u3)
	fmt.Printf("End of test\n")
}