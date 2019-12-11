package candles_test

import (
	"context"
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/plugins/candles"
	"code.vegaprotocol.io/vega/plugins/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
)

type candleTst struct {
	*candles.Candle
	ctx    context.Context
	cfunc  context.CancelFunc
	ctrl   *gomock.Controller
	store  *mocks.MockCandleStore
	trade  *mocks.MockTradeSub
	market *mocks.MockMarketSub
	tCh    chan []types.Trade
	mCh    chan []types.Market
}

func TestStartStopCandle(t *testing.T) {
	candle := getTestCandle(t)
	defer candle.Finish()
	marketID := "test-market"
	candle.Stop()
	// we do NOT expect the stores to be called, because we've already stopped the plugin
	// first register the market, so the trades can't fail because they're referring to an unknown market
	candle.mCh <- []types.Market{
		{
			Id: marketID,
		},
	}
	candle.tCh <- []types.Trade{{MarketID: marketID}}
}

func TestCandleTrades(t *testing.T) {
	candle := getTestCandle(t)
	defer candle.Finish()
	marketID := "test-market"
	tmpErr := errors.New("new-market-err")
	// we're expecting this to be called for each new market, multiplied by the number of intervals
	// which currently is 6. If there's an error from the stores, then no harm done, so we'll just
	// pretend like we didn't find anything. Use waitgroup to ensure we've added the market before
	// ending this test
	wg := sync.WaitGroup{}
	wg.Add(6)
	candle.store.EXPECT().FetchLastCandle(marketID, gomock.Any()).Times(6).Return(nil, tmpErr).Do(func(_ string, _ types.Interval) {
		wg.Done()
	})
	candle.mCh <- []types.Market{
		{
			Id: marketID,
		},
	}
	wg.Wait()
	// create candle for this new trade
	candle.tCh <- []types.Trade{{MarketID: marketID}}
	// this one doesn't create a candle, because the market is unkknown:
	// candle.tCh <- []types.Trade{{MarketID: "foobar-unknown-market"}}
}

func getTestCandle(t *testing.T) *candleTst {
	ctrl := gomock.NewController(t)
	trade := mocks.NewMockTradeSub(ctrl)
	market := mocks.NewMockMarketSub(ctrl)
	store := mocks.NewMockCandleStore(ctrl)
	tCh := make(chan []types.Trade, 1)
	mCh := make(chan []types.Market, 1)
	ctx, cfunc := context.WithCancel(context.Background())
	// expect Done call:
	trade.EXPECT().Done().AnyTimes().DoAndReturn(func() <-chan struct{} {
		return ctx.Done()
	})
	trade.EXPECT().Recv().AnyTimes().Return(tCh)
	market.EXPECT().Recv().AnyTimes().Return(mCh)
	return &candleTst{
		Candle: candles.NewCandle(ctx, store, trade, market),
		ctx:    ctx,
		cfunc:  cfunc,
		ctrl:   ctrl,
		store:  store,
		trade:  trade,
		market: market,
		tCh:    tCh,
		mCh:    mCh,
	}
}

func (t *candleTst) Finish() {
	t.cfunc()
	// ensure context cancel signal is there
	<-t.ctx.Done()
	t.ctrl.Finish()
	close(t.tCh)
	close(t.mCh)
}
