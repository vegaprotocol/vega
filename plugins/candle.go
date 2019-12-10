package plugins

import (
	"context"
	"fmt"
	"time"

	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

// CandleStore persistence for candles
//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_store_mock.go -package mocks code.vegaprotocol.io/vega/plugins CandleStore
type CandleStore interface {
	FetchLastCandle(marketID string, interval types.Interval) (*types.Candle, error)
	GenerateCandlesFromBuffer(marketID string, previousCandlesBuf map[string]types.Candle) error
}

// TradeSub subscription to the trade buffer
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_sub_mock.go -package mocks code.vegaprotocol.io/vega/plugins TradeSub
type TradeSub interface {
	Recv() <-chan []types.Trade
	Done() <-chan struct{}
}

// MarketSub subscription for the candles plugin to be aware of (new) markets
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_sub_mock.go -package mocks code.vegaprotocol.io/vega/plugins MarketSub
type MarketSub interface {
	Recv() <-chan []types.Market
	Done() <-chan struct{}
}

type Candle struct {
	store     CandleStore
	tradeSub  TradeSub
	mktSub    MarketSub
	buf       map[string]map[string]types.Candle
	stop      chan struct{}
	lastTrade types.Trade
}

// Currently we support 6 interval durations for trading candles on VEGA, as follows:
var supportedIntervals = [6]types.Interval{
	types.Interval_I1M,  // 1 minute
	types.Interval_I5M,  // 5 minutes
	types.Interval_I15M, // 15 minutes
	types.Interval_I1H,  // 1 hour
	types.Interval_I6H,  // 6 hours
	types.Interval_I1D,  // 1 day
}

// NewCandle creates a new candle plugin. The plugin immediately starts to listen for data coming
// from the subscriptions
func NewCandle(ctx context.Context, store CandleStore, tradeSub TradeSub, mktSub MarketSub) *Candle {
	cp := &Candle{
		store:    store,
		tradeSub: tradeSub,
		mktSub:   mktSub,
		buf:      map[string]map[string]types.Candle{},
		stop:     make(chan struct{}),
	}
	go cp.loop(ctx)
	return cp
}

func (c *Candle) Stop() {
	c.stop <- struct{}{}
}

func (c *Candle) loop(ctx context.Context) {
	defer func() {
		close(c.stop)
	}()
	for {
		select {
		case <-c.stop:
		case <-ctx.Done():
		case <-c.tradeSub.Done():
			return
		case trades := <-c.tradeSub.Recv():
			for _, trade := range trades {
				c.addTrade(trade)
			}
		case mkts := <-c.mktSub.Recv():
			ts := time.Now()
			for _, mkt := range mkts {
				c.start(mkt.Id, ts)
			}
		}
	}
}

func (c *Candle) start(marketID string, timestamp time.Time) {
	roundedTimestamps := getMapOfIntervalsToRoundedTimestamps(timestamp)
	previous := c.buf[marketID]
	c.buf[marketID] = map[string]types.Candle{}

	for _, interval := range supportedIntervals {
		bufkey := bufferKey(roundedTimestamps[interval], interval)
		var lastClose uint64
		if candl, ok := previous[bufkey]; ok {
			lastClose = candl.Close
		}

		if lastClose == 0 {
			previousCandle, err := c.store.FetchLastCandle(marketID, interval)
			if err == nil {
				lastClose = previousCandle.Close
			}
		}

		if lastClose == 0 {
			lastClose = c.lastTrade.Price
		}

		c.buf[marketID][bufkey] = newCandle(roundedTimestamps[interval], lastClose, 0, interval)
	}
}

func groupTrades(trades []types.Trade) map[string][]types.Trade {
	ret := map[string][]types.Trade{}
	maxCap := len(trades)
	for _, trade := range trades {
		mt, ok := ret[trade.MarketID]
		if !ok {
			mt = make([]types.Trade, 0, maxCap)
		}
		mt = append(mt, trade)
		// we've triaged another trade, one less left that could go to a different market
		maxCap--
	}
	return ret
}

// getMapOfIntervalsToRoundedTimestamps rounds timestamp to nearest minute, 5minute,
//  15 minute, hour, 6hours, 1 day intervals and return a map of rounded timestamps
func getMapOfIntervalsToRoundedTimestamps(timestamp time.Time) map[types.Interval]time.Time {
	timestamps := make(map[types.Interval]time.Time)

	// round floor by integer division
	for _, interval := range supportedIntervals {
		timestamps[interval] = vegatime.RoundToNearest(timestamp, interval)
	}

	return timestamps
}

// bufferKey returns the custom formatted buffer key for internal trade to timestamp mapping.
func bufferKey(timestamp time.Time, interval types.Interval) string {
	return fmt.Sprintf("%d:%s", timestamp.UnixNano(), interval.String())
}

// newCandle constructs a new candle with minimum required parameters.
func newCandle(timestamp time.Time, openPrice, size uint64, interval types.Interval) types.Candle {
	return types.Candle{
		Timestamp: timestamp.UnixNano(),
		Datetime:  vegatime.Format(timestamp),
		High:      openPrice,
		Low:       openPrice,
		Open:      openPrice,
		Close:     openPrice,
		Volume:    size,
		Interval:  interval,
	}
}

// AddTrade adds a trade to the trades buffer for the given market.
func (c *Candle) addTrade(trade types.Trade) {
	mktBuf, ok := c.buf[trade.MarketID]
	if !ok {
		// it's possible the market subscription expired, but we're getting trades from a new
		// market in already. In this case, we're just going to black-hole the trades in question
		// until we have a better understanding of what to do...
		return
	}
	for _, interval := range supportedIntervals {
		roundedTradeTime := vegatime.RoundToNearest(vegatime.UnixNano(trade.Timestamp), interval)

		bufkey := bufferKey(roundedTradeTime, interval)

		// check if bufferKey is present in buffer
		if candl, ok := mktBuf[bufkey]; ok {
			// if exists update the value of the candle under bufferKey with trade data
			updateCandle(&candl, &trade)
			mktBuf[bufkey] = candl
		} else {
			// if doesn't exist create new candle under this buffer key
			mktBuf[bufkey] = newCandle(roundedTradeTime, trade.Price, trade.Size, candl.Interval)
		}
		c.lastTrade = trade
	}
}

// updateCandle will calculate and set volume, open, close etc based on the given Trade.
func updateCandle(candle *types.Candle, trade *types.Trade) {
	// always overwrite close price
	candle.Close = trade.Price

	// candle.Volume == uint64(0) in case this is new candle and first trading activity happens for that candle !!!!
	// or candle.Open == uint64(0) in case there was no previous candle as this is a new market (aka also new trading activity for that candle)
	// -> overwrite open price with new trade price (by default candle.Open price is set to previous candle close price)
	// -> overwrite High and Low with new trade price (by default Low and High prices are set to candle open price which is set to previous candle close price)
	if candle.Volume == uint64(0) || candle.Open == uint64(0) {
		candle.Open = trade.Price
		candle.High = trade.Price
		candle.Low = trade.Price
	}

	// set minimum
	if trade.Price < candle.Low || candle.Low == uint64(0) {
		candle.Low = trade.Price
	}

	// set maximum
	if trade.Price > candle.High {
		candle.High = trade.Price
	}

	candle.Volume += trade.Size
}
