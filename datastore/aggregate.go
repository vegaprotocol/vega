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

func (store *memTradeStore) GetCandle(market string, sinceBlock, currentBlock uint64) (*msg.Candle, error) {
	if err := store.marketExists(market); err != nil {
		return nil, err
	}
	
	candle := &msg.Candle{
		CloseBlockNumber: currentBlock,
		OpenBlockNumber: sinceBlock,
	}

	for idx, t := range store.store.markets[market].tradesByTimestamp {
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
		if idx == len(store.store.markets[market].tradesByTimestamp)-1 {
			candle.Close = t.trade.Price
		}
	}

	return candle, nil
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
		candles[idx].CloseBlockNumber = candles[idx].OpenBlockNumber + interval - 1
	}

	found := false
	idx := 0

	for tidx, t := range store.store.markets[market].tradesByTimestamp {
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
				candles[idx].Close = store.store.markets[market].tradesByTimestamp[tidx-1].trade.Price
			}

			// if we start from a candle that is empty, and there are no previous candles to copy close price
			// its values should be populated with values of the previous trade that is outside of the sinceBlock scope
			if idx == 0 && tidx > 0 && candles[idx].Volume == 0 {
				candles[idx].Volume = 0
				candles[idx].Open = store.store.markets[market].tradesByTimestamp[tidx-1].trade.Price
				candles[idx].Close = store.store.markets[market].tradesByTimestamp[tidx-1].trade.Price
				candles[idx].High = store.store.markets[market].tradesByTimestamp[tidx-1].trade.Price
				candles[idx].Low = store.store.markets[market].tradesByTimestamp[tidx-1].trade.Price
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

func (m *memOrderStore) GetMarketDepth(market string) (*msg.MarketDepth, error) {
	if err := m.marketExists(market); err != nil {
		return &msg.MarketDepth{}, err
	}

	var (
		currentPrice uint64
		at uint64
		buysSideCumulative  uint64
		sellSideCumulative  uint64
	)

	orderBookDepth := msg.MarketDepth{Name: market, Buy: []*msg.PriceLevel{}, Sell: []*msg.PriceLevel{}}

	// repeat all twice for BUY side and SELL side
	// get all orders for market ordered by price
	// iterate through fetched orders and insert into a level
	// while iterating calculate cumulative volume and insert at consecutive price level
	for idx, order := range m.store.markets[market].buySideRemainingOrders.orders {
		if idx == 0 {
			orderBookDepth.Buy = append(orderBookDepth.Buy, &msg.PriceLevel{Price: order.price})
			currentPrice = order.price
		}

		if idx != 0 && currentPrice != order.price {
			currentPrice = order.price
			buysSideCumulative += orderBookDepth.Buy[at].Volume
			orderBookDepth.Buy[at].CumulativeVolume = buysSideCumulative
			orderBookDepth.Buy = append(orderBookDepth.Buy, &msg.PriceLevel{Price: order.price})
			at++
		}
		orderBookDepth.Buy[at].Volume += order.remaining
		orderBookDepth.Buy[at].NumberOfOrders++

		if idx + 1 == len(m.store.markets[market].buySideRemainingOrders.orders) {
			buysSideCumulative += orderBookDepth.Buy[at].Volume
			orderBookDepth.Buy[at].CumulativeVolume = buysSideCumulative
		}
	}

	currentPrice = 0
	at = 0
	for idx, order := range m.store.markets[market].sellSideRemainingOrders.orders {
		if idx == 0 {
			orderBookDepth.Sell = append(orderBookDepth.Sell, &msg.PriceLevel{Price: order.price})
			currentPrice = order.price
		}

		if idx != 0 && currentPrice != order.price {
			currentPrice = order.price
			sellSideCumulative += orderBookDepth.Sell[at].Volume
			orderBookDepth.Sell[at].CumulativeVolume = sellSideCumulative
			orderBookDepth.Sell = append(orderBookDepth.Sell, &msg.PriceLevel{Price: order.price})
			at++
		}
		orderBookDepth.Sell[at].Volume += order.remaining
		orderBookDepth.Sell[at].NumberOfOrders++

		if idx + 1 == len(m.store.markets[market].sellSideRemainingOrders.orders) {
			sellSideCumulative += orderBookDepth.Sell[at].Volume
			orderBookDepth.Sell[at].CumulativeVolume = sellSideCumulative
		}
	}

	return &orderBookDepth, nil
}
