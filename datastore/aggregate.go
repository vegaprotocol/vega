package datastore

import (
	"math"
	"vega/proto"
)

func newCandle() *msg.Candle {
	return &msg.Candle{}
}

func (store *memTradeStore) GetCandles(market string, sinceBlock, currentBlock, interval uint64) (msg.Candles, error) {
	if err := store.marketExists(market); err != nil {
		return msg.Candles{}, err
	}

	nOfCandles := uint64(math.Ceil(float64((currentBlock-sinceBlock)/interval)))+1
	var candles = make([]*msg.Candle, nOfCandles, nOfCandles)

	for idx := range candles {
		candles[idx] = &msg.Candle{}
		candles[idx].OpenBlockNumber = sinceBlock + uint64(idx) * interval
		candles[idx].CloseBlockNumber = candles[idx].OpenBlockNumber + interval
	}

	found := false
	idx := 0

	for tidx, t := range store.store.markets[market].tradesByTimestamp {
		// iterate trades until reached ones of interest
		if t.trade.Timestamp < sinceBlock {
			continue
		}

		// OK I have now only trades I need
		if candles[idx].OpenBlockNumber <= t.trade.Timestamp && t.trade.Timestamp < candles[idx].CloseBlockNumber {
			updateCandle(candles, idx, &t.trade)
		} else {
			// if current trade is not fit for current candle, close the candle with previous trade if non-empty candle
			if candles[idx].Volume != 0 {
				candles[idx].Close = store.store.markets[market].tradesByTimestamp[tidx-1].trade.Price
			}
			// proceed to next candle
			idx++
			// otherwise look for next candle that fits to the current trade and add update candle with new trade
			found = false
			for !found {
				// if reached the end of candles break
				if idx > int(nOfCandles)-1 {
					break
				}
				if candles[idx].OpenBlockNumber <= t.trade.Timestamp && t.trade.Timestamp < candles[idx].CloseBlockNumber {
					updateCandle(candles, idx, &t.trade)
					found = true
				} else {
					idx++
				}
			}
			// if reached the end of candles break
			if idx > int(nOfCandles)-1 {
				break
			}
		}
	}

	var output = msg.Candles{}
	output.Candles = candles
	return output, nil
}

func updateCandle(candles []*msg.Candle, idx int, trade *Trade) {
	if candles[idx].Volume == 0 {
		candles[idx].Open = trade.Price
	}
	candles[idx].Volume += trade.Size
	if candles[idx].High < trade.Price {
		candles[idx].High = trade.Price
	}
	if candles[idx].Low > trade.Price || candles[idx].Low == 0 {
		candles[idx].Low = trade.Price
	}
	//fmt.Printf("updated: %+v\n", candles[idx])
}

