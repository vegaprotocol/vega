package subscribers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

// CandleStore ...
type CandleStore interface {
	FetchLastCandle(marketID string, interval types.Interval) (*types.Candle, error)
	GenerateCandlesFromBuffer(marketID string, previousCandlesBuf map[string]types.Candle) error
}

type MarketTickEvt interface {
	events.Event
	MarketID() string
	Time() time.Time
}

type tradeBlock struct {
	trades []types.Trade
	last   types.Trade
	time   time.Time
	mID    string
}

type CandleSub struct {
	*Base
	store   CandleStore
	mu      sync.Mutex
	b2      map[string][]types.Trade
	last    map[string]types.Trade
	buf     []types.Trade
	tCh     chan tradeBlock
	candles map[string]map[string]types.Candle
}

// Currently we support 6 interval durations for trading candles on VEGA, as follows:
var supportedIntervals = [6]types.Interval{
	types.Interval_INTERVAL_I1M,  // 1 minute
	types.Interval_INTERVAL_I5M,  // 5 minutes
	types.Interval_INTERVAL_I15M, // 15 minutes
	types.Interval_INTERVAL_I1H,  // 1 hour
	types.Interval_INTERVAL_I6H,  // 6 hours
	types.Interval_INTERVAL_I1D,  // 1 day
}

func NewCandleSub(ctx context.Context, store CandleStore, ack bool) *CandleSub {
	sub := &CandleSub{
		Base:    NewBase(ctx, 1, ack),
		store:   store,
		b2:      map[string][]types.Trade{},
		last:    map[string]types.Trade{},
		buf:     []types.Trade{},
		tCh:     make(chan tradeBlock, 10), // ensure we're one block behind
		candles: map[string]map[string]types.Candle{},
	}
	go sub.internalLoop()
	return sub
}

func (c *CandleSub) internalLoop() {
	for {
		select {
		case <-c.Closed():
			return
		case block := <-c.tCh:
			// if no new trades, just check if we need to add the market ID to candles map
			if len(block.trades) == 0 {
				if _, ok := c.candles[block.mID]; !ok {
					c.candles[block.mID] = map[string]types.Candle{}
				}
			} else if len(block.trades) > 0 {
				c.updateCandles(block)
			}
		}
	}
}

func (c *CandleSub) Push(e events.Event) {
	switch te := e.(type) {
	case TE:
		trade := te.Trade()
		mID := trade.MarketID
		c.mu.Lock()
		if _, ok := c.b2[mID]; !ok {
			c.b2[mID] = []types.Trade{}
		}
		c.b2[mID] = append(c.b2[mID], trade)
		c.last[mID] = trade
		c.mu.Unlock()
	case NME:
		mID := te.Market().Id
		c.mu.Lock()
		if _, ok := c.b2[mID]; !ok {
			c.b2[mID] = []types.Trade{}
		}
		c.mu.Unlock()
		c.tCh <- tradeBlock{
			mID: mID,
		}
	case MarketTickEvt:
		mID := te.MarketID()
		c.mu.Lock()
		cpy := c.b2[mID]
		last := c.last[mID]
		c.b2[mID] = make([]types.Trade, 0, cap(cpy))
		c.mu.Unlock()
		c.tCh <- tradeBlock{
			trades: cpy,
			time:   te.Time(),
			last:   last,
			mID:    mID,
		}
	}
}

func (c *CandleSub) Types() []events.Type {
	return []events.Type{
		events.TradeEvent,
		events.MarketCreatedEvent,
		events.MarketTickEvent,
	}
}

func (c *CandleSub) updateCandles(block tradeBlock) {
	// Add trades and create candles
	lastByMarket := map[string]types.Trade{}
	for _, t := range block.trades {
		mID := t.MarketID
		for _, interval := range supportedIntervals {
			roundedTradeTime := vegatime.RoundToNearest(vegatime.UnixNano(t.Timestamp), interval)

			bufkey := bufferKey(roundedTradeTime, interval)

			// check if bufferKey is present in buffer
			mktBuf, ok := c.candles[mID]
			if !ok {
				mktBuf = map[string]types.Candle{}
				c.candles[mID] = mktBuf
			}
			if candl, ok := mktBuf[bufkey]; ok {
				// if exists update the value of the candle under bufferKey with trade data
				updateCandle(&candl, &t)
				mktBuf[bufkey] = candl
			} else {
				// if doesn't exist create new candle under this buffer key
				mktBuf[bufkey] = newCandle(roundedTradeTime, t.Price, t.Size, interval)
			}
			lastByMarket[mID] = t
		}
	}

	// Start logic (actually set last candles)
	roundedTimestamps := GetMapOfIntervalsToRoundedTimestamps(block.time)
	for mID, t := range lastByMarket {
		previous := c.candles[mID]
		for _, interval := range supportedIntervals {
			bufkey := bufferKey(roundedTimestamps[interval], interval)
			var lastClose uint64
			if candl, ok := previous[bufkey]; ok {
				lastClose = candl.Close
			}

			if lastClose == 0 {
				previousCandle, err := c.store.FetchLastCandle(mID, interval)
				if err == nil {
					lastClose = previousCandle.Close
				}
			}

			if lastClose == 0 {
				lastClose = t.Price
			}

			c.candles[mID][bufkey] = newCandle(roundedTimestamps[interval], lastClose, 0, interval)
		}
		_ = c.store.GenerateCandlesFromBuffer(mID, previous)
	}
}

// GetMapOfIntervalsToRoundedTimestamps rounds timestamp to nearest minute, 5minute,
//  15 minute, hour, 6hours, 1 day intervals and return a map of rounded timestamps
func GetMapOfIntervalsToRoundedTimestamps(timestamp time.Time) map[types.Interval]time.Time {
	timestamps := map[types.Interval]time.Time{}

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
