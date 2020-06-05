package candles_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/candles"
	"code.vegaprotocol.io/vega/candles/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testService struct {
	*candles.Svc
	ctx   context.Context
	cfunc context.CancelFunc
	ctrl  *gomock.Controller
	store *mocks.MockCandleStore
	log   *logging.Logger
}

type itMatcher struct {
	market   string
	interval types.Interval
	// ref      uint64
}

func getTestService(t *testing.T) *testService {
	ctrl := gomock.NewController(t)
	ctx, cfunc := context.WithCancel(context.Background())
	store := mocks.NewMockCandleStore(ctrl)
	log := logging.NewTestLogger()
	// create service, pass in mocks, ignore error
	svc, err := candles.NewService(
		log,
		candles.NewDefaultConfig(),
		store,
	)
	if err != nil {
		t.Fatalf("Unexpected error getting candle service: %+v", err)
	}
	return &testService{
		Svc:   svc,
		ctx:   ctx,
		cfunc: cfunc,
		ctrl:  ctrl,
		store: store,
		log:   log,
	}
}

func TestObserveCandles(t *testing.T) {
	t.Run("Observe candles - empty subscriptions", testObserveCandleStoreEmpty)
	t.Run("Observe candles - read values from channels", testObserveCandleStoreGetCandles)
	t.Run("Observe candles - ensure retry limit behaves as expected", testObserveCandlesRetries)
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
		types.Interval_INTERVAL_I1M,
		types.Interval_INTERVAL_I5M,
		types.Interval_INTERVAL_I15M,
		types.Interval_INTERVAL_I1H,
		types.Interval_INTERVAL_I6H,
		types.Interval_INTERVAL_I1D,
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
		types.Interval_INTERVAL_I1M,
		types.Interval_INTERVAL_I5M,
		types.Interval_INTERVAL_I15M,
		types.Interval_INTERVAL_I1H,
		types.Interval_INTERVAL_I6H,
		types.Interval_INTERVAL_I1D,
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
			svc.store.EXPECT().Subscribe(itMatcher{market: market, interval: it}).Times(1).Return(ref).Do(func(it *storage.InternalTransport) {
				candles, ok := expectedCandles[it.Market]
				assert.True(t, ok)
				for _, c := range candles {
					if c.Interval == it.Interval {
						// ensure the candle is pushed onto the channel
						wg.Add(1)
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
			ch, id := svc.ObserveCandles(svc.ctx, 10, &market, &it)
			c := <-ch
			assert.Equal(t, ref, id)
			// with a second wg, we could do this concurrently, but this is just a test...
			if c == nil {
				t.Fatalf("Failed to receive an observed candle")
			}
			assert.Equal(t, it, c.Interval)
			assert.Equal(t, ref, c.Open)
		}
	}
	svc.cfunc() // cancel context, we've made all the calls we needed to make, let's wait for unsubscribe calls to complete
	wg.Wait()
}

func testObserveCandlesRetries(t *testing.T) {
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
		types.Interval_INTERVAL_I1M,
		types.Interval_INTERVAL_I5M,
		types.Interval_INTERVAL_I15M,
		types.Interval_INTERVAL_I1H,
		types.Interval_INTERVAL_I6H,
		types.Interval_INTERVAL_I1D,
	}
	chWrite := func(ch chan<- *types.Candle) {
		// an empty literal is all we need
		ch <- &types.Candle{}
	}
	for f, market := range markets {
		// set up expected calls
		factor := f * len(intervals) // either 6 or len of intervals
		for i, it := range intervals {
			ref := uint64(i + 1 + factor)
			wg.Add(1)
			// in this test, we're not using the ready channel, because our goal is specifically to write to a channel that isn't being read
			svc.store.EXPECT().Subscribe(itMatcher{market: market, interval: it}).Times(1).DoAndReturn(func(it *storage.InternalTransport) uint64 {
				go chWrite(it.Transport)
				return ref
			})
			// ensure the same reference is unsubscribed when context is cancelled
			svc.store.EXPECT().Unsubscribe(ref).Times(1).Return(nil).Do(func(_ uint64) {
				wg.Done()
			})
			// we're setting a retry limit here, ignore the channel, this is all about retries
			_, id := svc.ObserveCandles(svc.ctx, 1, &market, &it)
			assert.Equal(t, ref, id)
		}
	}
	wg.Wait()
	svc.cfunc() // cancel context, we've made all the calls we needed to make, let's wait for unsubscribe calls to complete
}

func isSubscriptionEmpty(transport <-chan *types.Candle) bool {
	select {
	case <-transport:
		return false
	default:
		return true
	}
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
