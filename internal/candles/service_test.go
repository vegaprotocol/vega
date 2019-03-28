package candles

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/internal/candles/newmocks"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testService struct {
	Service
	ctx   context.Context
	cfunc context.CancelFunc
	ctrl  *gomock.Controller
	store *newmocks.MockCandleStore
	log   *logging.Logger
}

type itMatcher struct {
	market   string
	interval types.Interval
	ref      uint64
}

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

func getTestService(t *testing.T) *testService {
	ctrl := gomock.NewController(t)
	ctx, cfunc := context.WithCancel(context.Background())
	store := newmocks.NewMockCandleStore(ctrl)
	log := logging.NewLoggerFromEnv("dev")
	// create service, pass in mocks, ignore error
	svc, err := NewCandleService(
		NewDefaultConfig(log),
		store,
	)
	if err != nil {
		t.Fatalf("Unexpected error getting candle service: %+v", err)
	}
	return &testService{
		Service: svc,
		ctx:     ctx,
		cfunc:   cfunc,
		ctrl:    ctrl,
		store:   store,
		log:     log,
	}
}

func TestObserveCandles(t *testing.T) {
	t.Run("Observe candles - empty subscriptions", testObserveCandleStoreEmpty)
	t.Run("Observe candles - read values from channels", testObserveCandleStoreGetCandles)
}

func testObserveCandleStoreEmpty(t *testing.T) {
	svc := getTestService(t)
	// cancels context, syncs log, and finishes test controller
	defer svc.Finish()
	// wg ensuring unsubscribe was called when we expected it to be
	wg := sync.WaitGroup{}
	markets := []string{
		"BTC/DEC19",
		"ETH/APR19",
	}
	intervals := []types.Interval{
		types.Interval_I1M,
		types.Interval_I5M,
		types.Interval_I15M,
		types.Interval_I1H,
		types.Interval_I6H,
		types.Interval_I1D,
	}
	for f, market := range markets {
		// set up expected calls
		factor := f * len(intervals) // either 6 or len of intervals
		for i, it := range intervals {
			ref := uint64(i + 1 + factor)
			wg.Add(1)
			svc.store.EXPECT().Subscribe(itMatcher{market: market, interval: it}).Times(1).Return(ref)
			// ensure the same reference is unsubscribed when context is cancelled
			svc.store.EXPECT().Unsubscribe(ref).Times(1).Return(nil).Do(func(_ uint64) {
				wg.Done()
			})
			ch, id := svc.ObserveCandles(svc.ctx, 0, &market, &it)
			assert.Equal(t, ref, id)
			assert.True(t, isSubscriptionEmpty(ch))
		}
	}
	svc.cfunc() // cancel context, we've made all the calls we needed to make, let's wait for unsubscribe calls to complete
	wg.Wait()
}

func testObserveCandleStoreGetCandles(t *testing.T) {
	svc := getTestService(t)
	// cancels context, syncs log, and finishes test controller
	defer svc.Finish()
	// wg ensuring unsubscribe was called when we expected it to be
	wg := sync.WaitGroup{}
	markets := []string{
		"BTC/DEC19",
		"ETH/APR19",
	}
	intervals := []types.Interval{
		types.Interval_I1M,
		types.Interval_I5M,
		types.Interval_I15M,
		types.Interval_I1H,
		types.Interval_I6H,
		types.Interval_I1D,
	}
	expectedCandles := map[string][]*types.Candle{
		markets[0]: make([]*types.Candle, 0, len(intervals)),
		markets[1]: make([]*types.Candle, 0, len(intervals)),
	}
	for f, market := range markets {
		// set up expected calls
		factor := f * len(intervals) // either 6 or len of intervals
		for i, it := range intervals {
			ref := uint64(i + 1 + factor)
			expectedCandles[market] = append(expectedCandles[market], &types.Candle{
				Open:     ref,
				Interval: it,
			})
			wg.Add(1)
			svc.store.EXPECT().Subscribe(itMatcher{market: market, interval: it}).Times(1).Return(ref).Do(func(it *storage.InternalTransport) {
				candles, ok := expectedCandles[it.Market]
				assert.True(t, ok)
				for _, c := range candles {
					if c.Interval == it.Interval {
						// ensure the candle is pushed onto the channel
						go func(it *storage.InternalTransport) {
							it.Transport <- c
						}(it)
					}
				}
			})
			// ensure the same reference is unsubscribed when context is cancelled
			svc.store.EXPECT().Unsubscribe(ref).Times(1).Return(nil).Do(func(_ uint64) {
				wg.Done()
			})
			ch, id := svc.ObserveCandles(svc.ctx, 0, &market, &it)
			assert.Equal(t, ref, id)
			// with a second wg, we could do this concurrently, but this is just a test...
			c := <-ch
			assert.Equal(t, it, c.Interval)
			assert.Equal(t, ref, c.Open)
		}
	}
	svc.cfunc() // cancel context, we've made all the calls we needed to make, let's wait for unsubscribe calls to complete
	wg.Wait()
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
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
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

	candlesSubscription5mBTC, ref := candleService.ObserveCandles(ctx, 0, &MarketBTC, &interval5m)
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

func (t *testService) Finish() {
	t.cfunc()
	t.log.Sync()
	t.ctrl.Finish()
}

func (m itMatcher) String() string {
	return fmt.Sprintf("Market %s interval %v", m.market, m.interval)
}

func (m itMatcher) Matches(x interface{}) bool {
	var v storage.InternalTransport
	switch val := x.(type) {
	case *storage.InternalTransport:
		v = *val
	case storage.InternalTransport:
		v = val
	default:
		return false
	}
	return (v.Market == m.market && v.Interval == m.interval)
}
