package candles

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"

	"github.com/stretchr/testify/assert"
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
	candleStore, err := storage.NewCandleStore(storeConfig)
	assert.Nil(t, err)
	defer candleStore.Close()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	candleConfig := NewDefaultConfig(logger)
	candleService, err := NewCandleService(candleConfig, candleStore)
	assert.Nil(t, err)

	interval1m := types.Interval_I1M
	interval5m := types.Interval_I5M
	interval15m := types.Interval_I15M
	interval1h := types.Interval_I1H
	interval6h := types.Interval_I6H
	interval1d := types.Interval_I1D

	// -------- BTC MARKET SUBSCRIPTIONS -----

	candlesSubscription1mBTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval1m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1mBTC))
	assert.Equal(t, uint64(1), ref)

	candlesSubscription5mBTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval5m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription5mBTC))
	assert.Equal(t, uint64(2), ref)

	candlesSubscription15mBTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval15m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription15mBTC))
	assert.Equal(t, uint64(3), ref)

	candlesSubscription1hBTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval1h)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1hBTC))
	assert.Equal(t, uint64(4), ref)

	candlesSubscription6hBTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval6h)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription6hBTC))
	assert.Equal(t, uint64(5), ref)

	candlesSubscription1dBTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval1d)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1dBTC))
	assert.Equal(t, uint64(6), ref)

	// -------- ETH MARKET SUBSCRIPTIONS -----

	candlesSubscription1mETH, ref := candleService.ObserveCandles(ctx, &MarketETH, &interval1m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1mETH))
	assert.Equal(t, uint64(7), ref)

	candlesSubscription5mETH, ref := candleService.ObserveCandles(ctx, &MarketETH, &interval5m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription5mETH))
	assert.Equal(t, uint64(8), ref)

	candlesSubscription15mETH, ref := candleService.ObserveCandles(ctx, &MarketETH, &interval15m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription15mETH))
	assert.Equal(t, uint64(9), ref)

	candlesSubscription1hETH, ref := candleService.ObserveCandles(ctx, &MarketETH, &interval1h)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1hETH))
	assert.Equal(t, uint64(10), ref)

	candlesSubscription6hETH, ref := candleService.ObserveCandles(ctx, &MarketETH, &interval6h)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription6hETH))
	assert.Equal(t, uint64(11), ref)

	candlesSubscription1dETH, ref := candleService.ObserveCandles(ctx, &MarketETH, &interval1d)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription1dETH))
	assert.Equal(t, uint64(12), ref)

	// t0 = 2018-11-13T11:01:14Z
	t0 := uint64(1542106874000000000)

	go func() {
		for {
			select {
			case candle := <-candlesSubscription1mBTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0-uint64(14*time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, types.Interval_I1M, candle.Interval)

			case candle := <-candlesSubscription5mBTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0-uint64(time.Minute+14*time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, types.Interval_I5M, candle.Interval)

			case candle := <-candlesSubscription15mBTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0-uint64(time.Minute+14*time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, types.Interval_I15M, candle.Interval)

			case candle := <-candlesSubscription1hBTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0-uint64(time.Minute+14*time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, types.Interval_I1H, candle.Interval)

			case candle := <-candlesSubscription6hBTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0-uint64(5*time.Hour+time.Minute+14*time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, types.Interval_I6H, candle.Interval)

			case candle := <-candlesSubscription1dBTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0-uint64(11*time.Hour+time.Minute+14*time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, types.Interval_I1D, candle.Interval)

			case candle := <-candlesSubscription1mETH:
				fmt.Printf("RECEIVED CANDLE ETH %+v\n", candle)
				assert.Equal(t, t0-uint64(14*time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, types.Interval_I1M, candle.Interval)

			case candle := <-candlesSubscription5mETH:
				fmt.Printf("RECEIVED CANDLE ETH %+v\n", candle)
				assert.Equal(t, t0-uint64(time.Minute+14*time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, types.Interval_I5M, candle.Interval)

			case candle := <-candlesSubscription15mETH:
				fmt.Printf("RECEIVED CANDLE ETH %+v\n", candle)
				assert.Equal(t, t0-uint64(time.Minute+14*time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, types.Interval_I15M, candle.Interval)

			case candle := <-candlesSubscription1hETH:
				fmt.Printf("RECEIVED CANDLE ETH %+v\n", candle)
				assert.Equal(t, t0-uint64(time.Minute+14*time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, types.Interval_I1H, candle.Interval)

			case candle := <-candlesSubscription6hETH:
				fmt.Printf("RECEIVED CANDLE ETH %+v\n", candle)
				assert.Equal(t, t0-uint64(5*time.Hour+time.Minute+14*time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, types.Interval_I6H, candle.Interval)

			case candle := <-candlesSubscription1dETH:
				fmt.Printf("RECEIVED CANDLE ETH %+v\n", candle)
				assert.Equal(t, t0-uint64(11*time.Hour+time.Minute+14*time.Second), candle.Timestamp)
				assert.Equal(t, uint64(200), candle.Volume)
				assert.Equal(t, types.Interval_I1D, candle.Interval)

			}
		}
	}()

	var trades = []*types.Trade{
		{Id: "1", Market: MarketBTC, Price: uint64(100), Size: uint64(100), Timestamp: t0},
		{Id: "2", Market: MarketETH, Price: uint64(100), Size: uint64(100), Timestamp: t0},
		{Id: "3", Market: MarketBTC, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(20*time.Second)},
		{Id: "4", Market: MarketETH, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(20*time.Second)},
	}

	err = candleStore.StartNewBuffer(MarketBTC, t0)
	assert.Nil(t, err)
	err = candleStore.StartNewBuffer(MarketETH, t0)
	assert.Nil(t, err)
	for idx := range trades {
		err := candleStore.AddTradeToBuffer(*trades[idx])
		assert.Nil(t, err)
	}
	err = candleStore.GenerateCandlesFromBuffer(MarketBTC)
	assert.Nil(t, err)
	err = candleStore.GenerateCandlesFromBuffer(MarketETH)
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)
	fmt.Printf("End of test\n")
}

func isSubscriptionEmpty(transport <-chan *types.Candle) bool {
	select {
	case <-transport:
		return false
	default:
		return true
	}
}

func TestSubscriptionUpdates_MinMax(t *testing.T) {
	var wg sync.WaitGroup
	MarketBTC := "BTC/DEC19"
	var ctx = context.Background()
	storeConfig := storageConfig()
	storage.FlushStores(storeConfig)
	candleStore, err := storage.NewCandleStore(storeConfig)
	assert.Nil(t, err)
	defer candleStore.Close()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	candleConfig := NewDefaultConfig(logger)
	candleService, err := NewCandleService(candleConfig, candleStore)
	assert.Nil(t, err)

	interval5m := types.Interval_I5M

	candlesSubscription5mBTC, ref := candleService.ObserveCandles(ctx, &MarketBTC, &interval5m)
	assert.Equal(t, true, isSubscriptionEmpty(candlesSubscription5mBTC))
	assert.Equal(t, uint64(1), ref)

	// t0 = 2018-11-13T11:00:00Z
	t0 := uint64(1542106800000000000)

	// first update
	var trades1 = []*types.Trade{
		{Id: "1", Market: MarketBTC, Price: uint64(100), Size: uint64(100), Timestamp: t0},
		{Id: "2", Market: MarketBTC, Price: uint64(99), Size: uint64(100), Timestamp: t0 + uint64(10*time.Second)},
		{Id: "3", Market: MarketBTC, Price: uint64(108), Size: uint64(100), Timestamp: t0 + uint64(20*time.Second)},
		{Id: "4", Market: MarketBTC, Price: uint64(105), Size: uint64(100), Timestamp: t0 + uint64(30*time.Second)},
	}

	// second update
	var trades2 = []*types.Trade{
		{Id: "5", Market: MarketBTC, Price: uint64(110), Size: uint64(100), Timestamp: t0 + uint64(1*time.Minute)},
		{Id: "6", Market: MarketBTC, Price: uint64(112), Size: uint64(100), Timestamp: t0 + uint64(1*time.Minute+10*time.Second)},
		{Id: "7", Market: MarketBTC, Price: uint64(113), Size: uint64(100), Timestamp: t0 + uint64(1*time.Minute+20*time.Second)},
		{Id: "8", Market: MarketBTC, Price: uint64(109), Size: uint64(100), Timestamp: t0 + uint64(1*time.Minute+30*time.Second)},
	}

	// third update
	var trades3 = []*types.Trade{
		{Id: "9", Market: MarketBTC, Price: uint64(110), Size: uint64(100), Timestamp: t0 + uint64(2*time.Minute)},
		{Id: "10", Market: MarketBTC, Price: uint64(115), Size: uint64(100), Timestamp: t0 + uint64(2*time.Minute+10*time.Second)},
		{Id: "11", Market: MarketBTC, Price: uint64(90), Size: uint64(100), Timestamp: t0 + uint64(2*time.Minute+20*time.Second)},
		{Id: "12", Market: MarketBTC, Price: uint64(95), Size: uint64(100), Timestamp: t0 + uint64(2*time.Minute+30*time.Second)},
	}

	listenToCandles := func(wg *sync.WaitGroup, u1, u2, u3 *bool) {
		defer wg.Done()
		for {
			select {
			case candle := <-candlesSubscription5mBTC:
				fmt.Printf("RECEIVED CANDLE BTC %+v\n", candle)
				assert.Equal(t, t0, candle.Timestamp)
				assert.Equal(t, types.Interval_I5M, candle.Interval)

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
			if *u1 && *u2 && *u3 {
				break
			}
		}
	}

	var (
		u1, u2, u3 = false, false, false
	)
	wg.Add(1)
	go listenToCandles(&wg, &u1, &u2, &u3)

	// first update
	err = candleStore.StartNewBuffer(MarketBTC, t0)
	assert.Nil(t, err)
	for idx := range trades1 {
		err := candleStore.AddTradeToBuffer(*trades1[idx])
		assert.Nil(t, err)
	}
	err = candleStore.GenerateCandlesFromBuffer(MarketBTC)
	assert.Nil(t, err)

	// second update
	err = candleStore.StartNewBuffer(MarketBTC, t0+uint64(1*time.Minute))
	assert.Nil(t, err)
	for idx := range trades2 {
		err := candleStore.AddTradeToBuffer(*trades2[idx])
		assert.Nil(t, err)
	}
	err = candleStore.GenerateCandlesFromBuffer(MarketBTC)
	assert.Nil(t, err)

	// third update
	err = candleStore.StartNewBuffer(MarketBTC, t0+uint64(1*time.Minute))
	assert.Nil(t, err)
	for idx := range trades3 {
		err := candleStore.AddTradeToBuffer(*trades3[idx])
		assert.Nil(t, err)
	}
	err = candleStore.GenerateCandlesFromBuffer(MarketBTC)
	assert.Nil(t, err)

	wg.Wait()
	assert.True(t, u1)
	assert.True(t, u2)
	assert.True(t, u3)
	fmt.Printf("End of test\n")
}
