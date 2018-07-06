package datastore

import (
	"vega/proto"
)

type PriceHistory []*tradeInfo

type tradeInfo struct {
	timestamp uint64
	price     uint64
	size      uint64
}

func newCandle() *msg.Candle {
	return &msg.Candle{}
}

func (t *memTradeStore) GetCandles(market string, since, interval uint64) (candles msg.Candles, err error) {
	if err = t.marketExists(market); err != nil {
		return candles, err
	}

	var (
		currentTimestamp    uint64
		intervalProgression uint64
	)

	candle := newCandle()
	for idx, trade := range t.store.markets[market].priceHistory {
		// reach data slice for timestamps of interest
		if trade.timestamp < since {
			continue
		}

		// if new candle set open price of the first trade
		if candle.Open == 0 {
			candle.Open = trade.price
		}

		// check timestamp progression
		if currentTimestamp != trade.timestamp {
			currentTimestamp = trade.timestamp
			if candle.Volume != 0 {
				intervalProgression++
			}
		}

		// if in the interval adjust candle
		if intervalProgression < interval {
			candle.Volume += trade.size
			if candle.High < trade.price {
				candle.High = trade.price
			}
			if candle.Low > trade.price || candle.Low == 0 {
				candle.Low = trade.price
			}
		}

		// if reached the end of data finish
		if idx == len(t.store.markets[market].priceHistory)-1 {
			candle.Close = trade.price
			candles.Candles = append(candles.Candles, candle)
			break
		}

		// if reached end of interval and data in slice still available, append candle and progress
		if intervalProgression + 1 == interval &&
			t.store.markets[market].priceHistory[idx+1].timestamp != currentTimestamp {
			candle.Close = trade.price
			candles.Candles = append(candles.Candles, candle)
			intervalProgression = 0
			candle = newCandle()
		}
	}

	return candles, nil
}
