package datastore

import (
	"math"
	"vega/msg"
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

func (ts *memTradeStore) GetCandle(market string, sinceBlock, currentBlock uint64) (*msg.Candle, error) {
	if err := ts.marketExists(market); err != nil {
		return nil, err
	}
	
	candle := &msg.Candle{
		CloseBlockNumber: currentBlock,
		OpenBlockNumber: sinceBlock,
	}

	for idx, t := range ts.store.markets[market].tradesByTimestamp {
		// iterate trades until reached ones of interest
		if t.trade.Timestamp < sinceBlock {
			if t.trade.Price != 0 {
				// keep updating empty candle with latest price so that in case there are no trades of interest,
				// open close high and low values are set to the correct level of most recent trade
				candle.Open = t.trade.Price
				candle.Close = candle.Open
				candle.High = candle.Open
				candle.Low = candle.Open
				candle.Volume = 0
			}
			continue
		}

		if candle.Open == 0 {
			candle.Open = t.trade.Price
		}

		if candle.Volume == 0 {
			candle.Open = t.trade.Price
		}
		candle.Volume += t.trade.Size
		if candle.High < t.trade.Price {
			candle.High = t.trade.Price
		}
		if candle.Low > t.trade.Price || candle.Low == 0 {
			candle.Low = t.trade.Price
		}
		if idx == len(ts.store.markets[market].tradesByTimestamp)-1 {
			candle.Close = t.trade.Price
		}
	}

	return candle, nil
}

func (ts *memTradeStore) GetCandles(market string, sinceBlock, currentBlock, interval uint64) (msg.Candles, error) {
	if err := ts.marketExists(market); err != nil {
		return msg.Candles{}, err
	}

	nOfCandles := uint64(math.Ceil(float64((currentBlock-sinceBlock)/interval)))+1
	var candles = make([]*msg.Candle, nOfCandles, nOfCandles)

	for idx := range candles {
		candles[idx] = &msg.Candle{}
		candles[idx].OpenBlockNumber = sinceBlock + uint64(idx) * interval
		candles[idx].CloseBlockNumber = candles[idx].OpenBlockNumber + interval - 1
	}

	found := false
	idx := 0

	for tidx, t := range ts.store.markets[market].tradesByTimestamp {
		// iterate trades until reached ones of interest
		if t.trade.Timestamp < sinceBlock {
			continue
		}

		// OK I have now only trades I need
		if candles[idx].OpenBlockNumber <= t.trade.Timestamp && t.trade.Timestamp <= candles[idx].CloseBlockNumber {
			updateCandle(candles, idx, &t.trade)
		} else {
			// if current trade is not fit for current candle, close the candle with previous trade if non-empty candle
			if candles[idx].Volume != 0 {
				candles[idx].Close = ts.store.markets[market].tradesByTimestamp[tidx-1].trade.Price
			}

			// if we start from a candle that is empty, and there are no previous candles to copy close price
			// its values should be populated with values of the previous trade that is outside of the sinceBlock scope
			if idx == 0 && tidx > 0 && candles[idx].Volume == 0 {
				candles[idx].Volume = 0
				candles[idx].Open = ts.store.markets[market].tradesByTimestamp[tidx-1].trade.Price
				candles[idx].Close = ts.store.markets[market].tradesByTimestamp[tidx-1].trade.Price
				candles[idx].High = ts.store.markets[market].tradesByTimestamp[tidx-1].trade.Price
				candles[idx].Low = ts.store.markets[market].tradesByTimestamp[tidx-1].trade.Price
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
				if candles[idx].OpenBlockNumber <= t.trade.Timestamp && t.trade.Timestamp <= candles[idx].CloseBlockNumber {
					updateCandle(candles, idx, &t.trade)
					found = true
				} else {
					// if candle is empty apply values from previous one
					candles[idx].Volume = 0
					if idx >= 1 {
						candles[idx].Open = candles[idx-1].Close
						candles[idx].Close = candles[idx-1].Close
						candles[idx].High = candles[idx-1].Close
						candles[idx].Low = candles[idx-1].Close
					}
					idx++
				}
			}
			// if reached the end of candles break
			if idx > int(nOfCandles)-1 {
				break
			}
		}
		candles[idx].Close = t.trade.Price
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
}

func (m *orderStore) GetMarketDepth(market string) (*msg.MarketDepth, error) {
	if err := m.marketExists(market); err != nil {
		return &msg.MarketDepth{}, err
	}

	// get from store, recalculate accumulated volume and respond
	marketDepth := m.store.markets[market].marketDepth

	// recalculate accumulated volume
	for idx := range marketDepth.Buy {
		if idx == 0 {
			marketDepth.Buy[idx].CumulativeVolume = marketDepth.Buy[idx].Volume
			continue
		}
		marketDepth.Buy[idx].CumulativeVolume = marketDepth.Buy[idx-1].CumulativeVolume + marketDepth.Buy[idx].Volume
	}

	for idx := range marketDepth.Sell {
		if idx == 0 {
			marketDepth.Sell[idx].CumulativeVolume = marketDepth.Sell[idx].Volume
			continue
		}
		marketDepth.Sell[idx].CumulativeVolume = marketDepth.Sell[idx-1].CumulativeVolume + marketDepth.Sell[idx].Volume
	}

	orderBookDepth := msg.MarketDepth{Name: market, Buy: marketDepth.Buy, Sell: marketDepth.Sell}

	return &orderBookDepth, nil
}
