package datastore

import (
	"fmt"

	msg "vega/msg"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"time"
)

type MarketDepth struct {
	Name string
	Buy  []*msg.PriceLevel
	Sell []*msg.PriceLevel
}

type MarketDepthManager interface {
	updateWithRemaining(order *msg.Order)
	updateWithRemainingDelta(order *msg.Order, remainingDelta uint64)
	removeWithRemaining(order *msg.Order)
	getBuySide() []*msg.PriceLevel
	getSellSide() []*msg.PriceLevel
}

func NewMarketDepthUpdaterGetter() MarketDepthManager {
	return &MarketDepth{}
}

// recalculate cumulative volume only once when fetching the MarketDepth

func (md *MarketDepth) updateWithRemainingBuySide(order *msg.Order) {
	var at = -1

	for idx, priceLevel := range md.Buy {
		if priceLevel.Price > order.Price {
			continue
		}

		if priceLevel.Price == order.Price {
			// update price level
			md.Buy[idx].Volume += order.Remaining
			md.Buy[idx].NumberOfOrders++
			// updated - job done
			return
		}

		at = idx
		break
	}

	if at == -1 {
		// reached the end and not found, append at the end
		md.Buy = append(md.Buy, &msg.PriceLevel{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1})
		return
	}
	// found insert at
	md.Buy = append(md.Buy[:at], append([]*msg.PriceLevel{{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1}}, md.Buy[at:]...)...)
}

func (md *MarketDepth) updateWithRemainingSellSide(order *msg.Order) {
	var at = -1

	for idx, priceLevel := range md.Sell {
		if priceLevel.Price < order.Price {
			continue
		}

		if priceLevel.Price == order.Price {
			// update price level
			md.Sell[idx].Volume += order.Remaining
			md.Sell[idx].NumberOfOrders++
			// updated - job done
			return
		}

		at = idx
		break
	}

	if at == -1 {
		md.Sell = append(md.Sell, &msg.PriceLevel{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1})
		return
	}
	// found insert at
	md.Sell = append(md.Sell[:at], append([]*msg.PriceLevel{{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1}}, md.Sell[at:]...)...)
}

func (md *MarketDepth) updateWithRemaining(order *msg.Order) {
	if order.Side == msg.Side_Buy {
		md.updateWithRemainingBuySide(order)
	}
	if order.Side == msg.Side_Sell {
		md.updateWithRemainingSellSide(order)
	}
}

func (md *MarketDepth) updateWithRemainingDelta(order *msg.Order, remainingDelta uint64) {
	if order.Side == msg.Side_Buy {
		for idx, priceLevel := range md.Buy {
			if priceLevel.Price > order.Price {
				continue
			}

			if priceLevel.Price == order.Price {
				// update price level
				md.Buy[idx].Volume -= remainingDelta
				// updated - job done

				// safeguard - shouldn't happen but if volume for gets negative remove price level
				if md.Buy[idx].Volume <= 0 {
					copy(md.Buy[idx:], md.Buy[idx+1:])
					md.Buy = md.Buy[:len(md.Buy)-1]
				}
				return
			}
		}
		// not found
		return
	}

	if order.Side == msg.Side_Sell {
		for idx, priceLevel := range md.Sell {
			if priceLevel.Price < order.Price {
				continue
			}

			if priceLevel.Price == order.Price {
				// update price level
				md.Sell[idx].Volume -= remainingDelta
				// updated - job done

				// safeguard - shouldn't happen but if volume for gets negative remove price level
				if md.Sell[idx].Volume <= 0 {
					copy(md.Sell[idx:], md.Sell[idx+1:])
					md.Sell = md.Sell[:len(md.Sell)-1]
				}
				return
			}
		}
		// not found
		return
	}
}

func (md *MarketDepth) removeWithRemaining(order *msg.Order) {
	if order.Side == msg.Side_Buy {
		for idx, priceLevel := range md.Buy {
			if priceLevel.Price > order.Price {
				continue
			}

			if priceLevel.Price == order.Price {
				// update price level
				md.Buy[idx].NumberOfOrders--
				md.Buy[idx].Volume -= order.Remaining

				// remove empty price level
				if md.Buy[idx].NumberOfOrders == 0 || md.Buy[idx].Volume <= 0 {
					copy(md.Buy[idx:], md.Buy[idx+1:])
					md.Buy = md.Buy[:len(md.Buy)-1]
				}
				// updated - job done
				return
			}
		}
		// not found
		return
	}

	if order.Side == msg.Side_Sell {
		for idx, priceLevel := range md.Sell {
			if priceLevel.Price < order.Price {
				continue
			}

			if priceLevel.Price == order.Price {
				// update price level
				md.Sell[idx].NumberOfOrders--
				md.Sell[idx].Volume -= order.Remaining

				// remove empty price level
				if md.Sell[idx].NumberOfOrders == 0 || md.Sell[idx].Volume <= 0 {
					copy(md.Sell[idx:], md.Sell[idx+1:])
					md.Sell = md.Sell[:len(md.Sell)-1]
				}
				// updated - job done
				return
			}
		}
		// not found
		return
	}
}

func (md *MarketDepth) getBuySide() []*msg.PriceLevel {
	return md.Buy
}

func (md *MarketDepth) getSellSide() []*msg.PriceLevel {
	return md.Sell
}

type candleGenerator struct {
	market          string
	persistentStore *badger.DB
	tradesBuffer    []*msg.Trade
}

func NewCandleGenerator(market string) candleGenerator {
	return candleGenerator{market, nil, nil}
}

func (cp *candleGenerator) AddTrade(trade *msg.Trade) {
	cp.tradesBuffer = append(cp.tradesBuffer, trade)
}

// this should act as a separate slow go routine triggered after block is committed
func (cp *candleGenerator) Generate() error {

	for idx := range cp.tradesBuffer {
		// generate candle keys
		candleKeys := generateCandleKeysForTrade(cp.tradesBuffer[idx])

		// for each trade generate candle keys and run update on each bucket
		txn := cp.persistentStore.NewTransaction(true)
		for _, key := range candleKeys {

			item, err := txn.Get(key)

			if err == badger.ErrEmptyKey {
				candle := NewCandle(cp.tradesBuffer[idx].Price, cp.tradesBuffer[idx].Size)
				candleBuf, err := proto.Marshal(candle)
				if err != nil {
					return err
				}

				if err = txn.Set(key, candleBuf); err != nil {
					return err
				}
			}

			if err == nil {
				// umarshal fetched candle
				var candleForUpdate msg.Candle
				itemCopy, err := item.ValueCopy(nil)
				proto.Unmarshal(itemCopy, &candleForUpdate)

				// update fetched candle with new trade
				UpdateCandle(&candleForUpdate, cp.tradesBuffer[idx])

				// marshal candle
				candleBuf, err := proto.Marshal(&candleForUpdate)
				if err != nil {
					return err
				}

				// push candle to badger
				if err = txn.Set(key, candleBuf); err != nil {
					return err
				}
			}
		}

		if err := txn.Commit(); err != nil {
			return err
		}
	}

	if len(cp.tradesBuffer) == 0 {
		if err := cp.progressWithEmptyCandles(); err != nil {
			return err
		}
	}

	return nil
}

func (cp *candleGenerator) progressWithEmptyCandles() error {
	// if t is empty we need to take vegatime and update candles if necessary FOR ALL MARKETS

	currentvegatime := int64(1305861602)

	// generate keys for this timestamp

	candleKeys := generateCandleKeysForCurrentTimestamp(cp.market, currentvegatime)

	// if key does not exist seek most recent values, create empty candle with those close value and insert
	txn := cp.persistentStore.NewTransaction(true)

	// for all candle intervals
	for _, key := range candleKeys {

		// if key does not exist, seek most recent value
		_, err := txn.Get(key)
		if err == badger.ErrEmptyKey {
			prefixForMostRecent := append([]byte(string(key)[:len(string(key))-19]), 0xFF)
			options := badger.DefaultIteratorOptions
			options.Reverse = true
			it := txn.NewIterator(options)
			it.Seek(prefixForMostRecent)
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {

			}

			// extract close price from previous candle
			var previousCandle msg.Candle
			proto.Unmarshal(value, &previousCandle)

			// generate new candle with extracted close price
			newCandle := NewCandle(previousCandle.Close, 0)
			candleBuf, err := proto.Marshal(newCandle)
			if err != nil {
				return err
			}

			// push new candle to the
			if err := txn.Set(key, candleBuf); err != nil {
				return err
			}
		}
		//if present do nothing
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

func NewCandle(openPrice, size uint64) *msg.Candle {
	//TODO: get candle form pool of candles
	var candle *msg.Candle
	candle.Open = openPrice
	candle.Low = openPrice
	candle.High = openPrice
	candle.Close = openPrice
	candle.Volume = size
	return candle
}

func UpdateCandle(candle *msg.Candle, trade *msg.Trade) {
	// always overwrite close price
	candle.Close = trade.Price
	// set minimum
	if trade.Price < candle.Low {
		candle.Low = trade.Price
	}
	// set maximum
	if trade.Price > candle.High {
		candle.High = trade.Price
	}
	candle.Volume += trade.Size
}

func generateCandleKeysForTrade(trade *msg.Trade) map[string][]byte {
	keys := make(map[string][]byte)
	timestamps := getMapOfIntervalsToTimestamps(int64(trade.Timestamp))

	for key, val := range timestamps {
		keys[key] = []byte(fmt.Sprintf("M:%s_I:%s_T:%s", trade.Market, key, val))
	}

	return keys
}

func generateCandleKeysForCurrentTimestamp(market string, timestamp int64) map[string][]byte {
	keys := make(map[string][]byte)
	timestamps := getMapOfIntervalsToTimestamps(timestamp)

	for key, val := range timestamps {
		keys[key] = []byte(fmt.Sprintf("M:%s_I:%s_T:%s", market, key, val))
	}

	return keys
}

func getMapOfIntervalsToTimestamps(timestamp int64) map[string]string {
	timestamps := make(map[string]string)
	t := time.Unix(int64(1305861602), 0)

	roundedToMinute := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	fmt.Printf("roundedToMinute %+v\n", roundedToMinute)
	timestamps["1m"] = fmt.Sprintf("%d", roundedToMinute.UnixNano())

	fmt.Printf("\n%d\n", t.Minute())
	roundedTo5Minutes := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), (t.Minute()/5)*5, 0, 0, t.Location())
	fmt.Printf("roundedToMinute %+v\n", roundedTo5Minutes)
	timestamps["5m"] = fmt.Sprintf("%d", roundedTo5Minutes.UnixNano())

	roundedTo15Minutes := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), (t.Minute()/15)*15, 0, 0, t.Location())
	fmt.Printf("roundedTo15Minutes %+v\n", roundedTo15Minutes)
	timestamps["15m"] = fmt.Sprintf("%d", roundedTo15Minutes.UnixNano())

	roundedTo1Hour := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	fmt.Printf("roundedTo1Hour %+v\n", roundedTo1Hour)
	timestamps["1h"] = fmt.Sprintf("%d", roundedTo1Hour.UnixNano())

	roundedTo6Hour := time.Date(t.Year(), t.Month(), t.Day(), (t.Hour()/6)*6, 0, 0, 0, t.Location())
	fmt.Printf("roundedTo6Hour %+v\n", roundedTo6Hour)
	timestamps["6h"] = fmt.Sprintf("%d", roundedTo6Hour.UnixNano())

	roundedToDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	fmt.Printf("roundedToDay %+v\n", roundedToDay)
	timestamps["1d"] = fmt.Sprintf("%d", roundedToDay.UnixNano())

	return timestamps
}

// STEP 1
// DONE napisac algorytm wrzucania do generowania kluczy ktory na podstawie timestampu umie zaookraglic do najblizszej rownej wartosci

// STEP 2
// DONE dla kazdego ze zgenorwanych kluczy
// - wyciagnij wartosc, sparsuj do candle
// - zrob updejt na candlu z nowa wartoscia.
// - sparsuj do binarki i wstaw candla z powrotem

// STEP 3
// napisac ladny algortym generowania kluczy dla operacji fetch
// ktory parsuje since time do unix stampa i na podstawie interwalu (time.Second) szuka najblizszego matcha poprzez seek
