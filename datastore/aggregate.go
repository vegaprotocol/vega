package datastore

import (
	"math"
	"vega/services/msg"
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

	nOfCandles := uint64(math.Ceil(float64((currentBlock-sinceBlock)/interval))) + 1
	var candles = make([]*msg.Candle, nOfCandles, nOfCandles)

	for idx := range candles {
		candles[idx] = &msg.Candle{}
		candles[idx].OpenBlockNumber = sinceBlock + uint64(idx)*interval
		candles[idx].CloseBlockNumber = candles[idx].OpenBlockNumber + interval
	}

	found := false
	idx := 0

	for tidx, trade := range t.store.markets[market].priceHistory {
		// iterate trades until reached ones of interest
		if trade.timestamp < sinceBlock {
			continue
		}

		// OK I have now only trades I need
		if candles[idx].OpenBlockNumber <= trade.timestamp && trade.timestamp < candles[idx].CloseBlockNumber {
			updateCandle(candles, idx, trade)
		} else {
			// if current trade is not fit for current candle, close the candle with previous trade if non-empty candle
			if candles[idx].Volume != 0 {
				candles[idx].Close = t.store.markets[market].priceHistory[tidx-1].price
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
				if candles[idx].OpenBlockNumber <= trade.timestamp && trade.timestamp < candles[idx].CloseBlockNumber {
					updateCandle(candles, idx, trade)
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

func updateCandle(candles []*msg.Candle, idx int, trade *tradeInfo) {
	if candles[idx].Volume == 0 {
		candles[idx].Open = trade.price
	}
	candles[idx].Volume += trade.size
	if candles[idx].High < trade.price {
		candles[idx].High = trade.price
	}
	if candles[idx].Low > trade.price || candles[idx].Low == 0 {
		candles[idx].Low = trade.price
	}
	//fmt.Printf("updated: %+v\n", candles[idx])
}
