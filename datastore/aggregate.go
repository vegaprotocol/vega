package datastore

import (
	"fmt"
	"math"
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

func (t *memTradeStore) GetCandles(market string, sinceBlock, currentBlock, interval uint64) (msg.Candles, error) {
	if err := t.marketExists(market); err != nil {
		return msg.Candles{}, err
	}

	nOfCandles := uint64(math.Ceil(float64((currentBlock-sinceBlock)/interval)))
	var candles = make([]*msg.Candle, nOfCandles, nOfCandles)

	fmt.Printf("%d\n", nOfCandles)

	fmt.Printf("candles %+v\n", candles)

	for idx := range candles {
		candles[idx] = &msg.Candle{}
		fmt.Printf("%d %d %d\n", sinceBlock, idx, interval)
		fmt.Printf("%d %d %d\n", sinceBlock, uint64(idx), interval)
		candles[idx].OpenBlockNumber = sinceBlock + uint64(idx) * interval
		candles[idx].CloseBlockNumber = candles[idx].OpenBlockNumber + interval
	}

	fmt.Printf("candles with open close %+v\n", candles)
	for _, c := range candles {
		fmt.Printf("%+v\n", c)
	}


	idx := 0
	for tidx, trade := range t.store.markets[market].priceHistory {
		// iterate trades until reached ones of interest
		if trade.timestamp < sinceBlock {
			continue
		}

		// OK I have now only trades I need
		if candles[idx].OpenBlockNumber <= trade.timestamp && trade.timestamp < candles[idx].CloseBlockNumber {
			if candles[idx].Volume == 0 {
				candles[idx].Open = trade.price
			}
			candles[idx].Volume = trade.size
			if candles[idx].High < trade.price {
				candles[idx].High = trade.price
			}
			if candles[idx].Low > trade.price || candles[idx].Low == 0 {
				candles[idx].Low = trade.price
			}
		} else {
			candles[idx].Close = t.store.markets[market].priceHistory[tidx-1].price
			idx++
		}
		if len(candles) ==
	}

	var output = msg.Candles{}
	output.Candles = candles
	return output, nil
}

func (t *memTradeStore) GetCandles1(market string, since, interval uint64) (candles msg.Candles, err error) {
	if err = t.marketExists(market); err != nil {
		return candles, err
	}

	var (
		currentTimestamp    uint64
		intervalProgression uint64
	)

	fmt.Printf("t.store.markets[market].priceHistory: %+v\n", len(t.store.markets[market].priceHistory))

	candle := newCandle()
	for idx, trade := range t.store.markets[market].priceHistory {
		// reach data slice for timestamps of interest
		if trade.timestamp < since {
			continue
		}

		fmt.Printf("trade: %+v\n", trade)

		// if new candle set open price of the first trade
		if candle.Open == 0 {
			candle.Open = trade.price
		}

		// check timestamp progression
		if currentTimestamp != trade.timestamp {
			if candle.Volume != 0 {
				fmt.Printf("intervalProgression: %d\n", intervalProgression)
				intervalProgression += trade.timestamp - currentTimestamp
				fmt.Printf("intervalProgression: %d\n", intervalProgression)
				//intervalProgression++
			}
			currentTimestamp = trade.timestamp
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
			fmt.Printf("candle updated: %+v\n", candle)
		}

		// if reached the end of data finish
		if idx == len(t.store.markets[market].priceHistory)-1 {
			candle.Close = trade.price
			fmt.Printf("candle: %+v\n", candle)
			candles.Candles = append(candles.Candles, candle)
			break
		}

		// if reached end of interval and data in slice still available, append candle and progress
		if intervalProgression + 1 == interval &&
			t.store.markets[market].priceHistory[idx+1].timestamp != currentTimestamp {
			candle.Close = trade.price
			fmt.Printf("candle: %+v\n", candle)
			candles.Candles = append(candles.Candles, candle)
			intervalProgression = 0
			candle = newCandle()
			continue
		}

		// gap identified
		if intervalProgression + 1 > interval {
			candle.Close = trade.price
			fmt.Printf("candle: %+v\n", candle)
			candles.Candles = append(candles.Candles, candle)
			intervalProgression = 0
			candle = newCandle()
			candle.Volume += trade.size
			if candle.High < trade.price {
				candle.High = trade.price
			}
			if candle.Low > trade.price || candle.Low == 0 {
				candle.Low = trade.price
			}
			fmt.Printf("candle updated: %+v\n", candle)
			continue
		}
	}

	return candles, nil
}
