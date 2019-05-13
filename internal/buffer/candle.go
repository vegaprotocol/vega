package buffer

import (
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"
)

// Currently we support 6 interval durations for trading candles on VEGA, as follows:
var supportedIntervals = [6]types.Interval{
	types.Interval_I1M,  // 1 minute
	types.Interval_I5M,  // 5 minutes
	types.Interval_I15M, // 15 minutes
	types.Interval_I1H,  // 1 hour
	types.Interval_I6H,  // 6 hours
	types.Interval_I1D,  // 1 day
}

type CandleStore interface {
	FetchLastCandle(marketID string, interval types.Interval) (*types.Candle, error)
}

type Candle struct {
	buf       map[string]types.Candle
	marketID  string
	store     CandleStore
	mu        sync.Mutex
	lastTrade types.Trade
}

func NewCandle(marketID string, store CandleStore, now time.Time) *Candle {
	candl := &Candle{
		buf:      map[string]types.Candle{},
		marketID: marketID,
		store:    store,
	}

	candl.Start(now)
	return candl
}

func (c *Candle) reset() {
	c.buf = map[string]types.Candle{}
}

func (c *Candle) Start(timestamp time.Time) (map[string]types.Candle, error) {
	c.mu.Lock()
	roundedTimestamps := GetMapOfIntervalsToRoundedTimestamps(timestamp)
	previous := c.buf
	c.reset()

	for _, interval := range supportedIntervals {
		bufkey := bufferKey(roundedTimestamps[interval], interval)
		var lastClose uint64
		if candl, ok := previous[bufkey]; ok {
			lastClose = candl.Close
		}

		if lastClose == 0 {
			previousCandle, err := c.store.FetchLastCandle(c.marketID, interval)
			if err == nil {
				lastClose = previousCandle.Close
			}
		}

		if lastClose == 0 {
			lastClose = c.lastTrade.Price
		}

		c.buf[bufkey] = newCandle(roundedTimestamps[interval], lastClose, 0, interval)
	}

	c.mu.Unlock()
	return previous, nil
}

// AddTradeToBuffer adds a trade to the trades buffer for the given market.
func (c *Candle) AddTrade(trade types.Trade) error {
	for _, interval := range supportedIntervals {
		roundedTradeTime := vegatime.RoundToNearest(vegatime.UnixNano(trade.Timestamp), interval)

		bufkey := bufferKey(roundedTradeTime, interval)

		c.mu.Lock()
		// check if bufferKey is present in buffer
		if candl, ok := c.buf[bufkey]; ok {
			// if exists update the value of the candle under bufferKey with trade data
			updateCandle(&candl, &trade)
			c.buf[bufkey] = candl
		} else {
			// if doesn't exist create new candle under this buffer key
			c.buf[bufkey] = newCandle(roundedTradeTime, trade.Price, trade.Size, candl.Interval)
		}
		c.lastTrade = trade
		c.mu.Unlock()
	}

	return nil
}

// GetMapOfIntervalsToRoundedTimestamps rounds timestamp to nearest minute, 5minute,
//  15 minute, hour, 6hours, 1 day intervals and return a map of rounded timestamps
func GetMapOfIntervalsToRoundedTimestamps(timestamp time.Time) map[types.Interval]time.Time {
	timestamps := make(map[types.Interval]time.Time)

	// round floor by integer division
	for _, interval := range supportedIntervals {
		timestamps[interval] = vegatime.RoundToNearest(timestamp, interval)
	}

	return timestamps
}

// getBufferKey returns the custom formatted buffer key for internal trade to timestamp mapping.
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
