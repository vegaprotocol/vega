package datastore

import (
	"errors"
	"fmt"
	"sync"

	"vega/log"
	"vega/msg"
	"vega/vegatime"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
)

var supportedIntervals = [6]msg.Interval{
	msg.Interval_I1M, msg.Interval_I5M, msg.Interval_I15M, msg.Interval_I1H, msg.Interval_I6H, msg.Interval_I1D,}

type candleStore struct {
	badger *badgerStore

	subscribers  map[uint64]map[msg.Interval]chan msg.Candle
	buffer       map[msg.Interval]msg.Candle
	candleBuffer map[string]map[string]msg.Candle
	subscriberId uint64
	mu           sync.Mutex
}

func NewCandleStore(dir string) CandleStore {
	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	//opts.SyncWrites = true
	db, err := badger.Open(opts)
	if err != nil {
		fmt.Printf(err.Error())
	}
	bs := badgerStore{db: db}
	return &candleStore{badger: &bs, buffer: make(map[msg.Interval]msg.Candle),
		subscribers: make(map[uint64]map[msg.Interval]chan msg.Candle),
		candleBuffer: make(map[string]map[string]msg.Candle),
	}
}

func (c *candleStore) Subscribe(internalTransport map[msg.Interval]chan msg.Candle) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initialiseInternalTransport(internalTransport)

	c.subscriberId = c.subscriberId + 1
	c.subscribers[c.subscriberId] = internalTransport
	log.Debugf("CandleStore -> Subscribe: Candle subscriber added: %d", c.subscriberId)
	return c.subscriberId
}

func (c *candleStore) initialiseInternalTransport(internalTransport map[msg.Interval]chan msg.Candle) {
	internalTransport[msg.Interval_I1M] = make(chan msg.Candle, 1)
	internalTransport[msg.Interval_I5M] = make(chan msg.Candle, 1)
	internalTransport[msg.Interval_I15M] = make(chan msg.Candle, 1)
	internalTransport[msg.Interval_I1H] = make(chan msg.Candle, 1)
	internalTransport[msg.Interval_I6H] = make(chan msg.Candle, 1)
	internalTransport[msg.Interval_I1D] = make(chan msg.Candle, 1)
}

func (c *candleStore) Unsubscribe(id uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.subscribers) == 0 {
		log.Debugf("CandleStore -> Unsubscribe: No subscribers connected")
		return nil
	}

	if _, exists := c.subscribers[id]; exists {
		delete(c.subscribers, id)
		log.Debugf("CandleStore -> Unsubscribe: Subscriber removed: %v", id)
		return nil
	}
	return errors.New(fmt.Sprintf("CandleStore subscriber does not exist with id: %d", id))
}

func (c *candleStore) Notify() error {

	if len(c.subscribers) == 0 {
		log.Debugf("CandleStore -> Notify: No subscribers connected")
		return nil
	}

	if c.buffer == nil {
		// Only publish when we have items
		log.Debugf("CandleStore -> Notify: No new candle")
		return nil
	}

	c.mu.Lock()
	intervalsToCandlesMap := c.buffer
	c.mu.Unlock()

	// update candle for each interval for each subscriber
	for id, internalTransport := range c.subscribers {
		for interval, candleForUpdate := range intervalsToCandlesMap {
			select {
			case internalTransport[interval] <- candleForUpdate:
				log.Debugf("Candle updated for interval: %s", interval)
				break
			default:
				log.Infof("Candles state could not been updated for subscriber %d at interval %s", id, interval)
			}
		}
	}
	return nil
}

// thread safe
func (c *candleStore) QueueEvent(candle msg.Candle, interval msg.Interval) error {

	c.buffer[interval] = candle

	log.Debugf("CandleStore -> queueEvent: Adding candle to buffer of intervals at: %s", interval)
	return nil
}

func (c *candleStore) StartNewBuffer(market string, timestamp uint64) {
	//fmt.Printf("\nStartNewBuffer called \n\n")
	roundedTimestamps := getMapOfIntervalsToRoundedTimestamps(timestamp)

	// set state
	previousCandleBuffer := c.candleBuffer[market]

	c.resetCandleBuffer(market);

	for _, interval := range supportedIntervals {
		bufferKey := getBufferKey(roundedTimestamps[interval], interval)
		lastClose := previousCandleBuffer[bufferKey].Close
		if lastClose == uint64(0) {
			prefixForMostRecent, _ := c.badger.candlePrefix(market, interval, true)
			txn := c.badger.db.NewTransaction(true)
			previousCandle, err := c.fetchMostRecentCandle(txn, prefixForMostRecent)
			if err != nil {
				lastClose = 0
			} else {
				lastClose = previousCandle.Close
			}
		}
		c.candleBuffer[market][bufferKey] = *NewCandle(roundedTimestamps[interval], lastClose, 0, interval)
	}

	//// should derive close price from the previous version
	//if _, ok := c.candleBuffer[market]; ok {
	//	for _, interval := range supportedIntervals {
	//		bufferKey := getBufferKey(roundedTimestamps[interval], interval)
	//		lastClose := previousCandleBuffer[bufferKey].Close
	//		c.candleBuffer[market][bufferKey] = *NewCandle(roundedTimestamps[interval], lastClose, 0, interval)
	//	}
	//} else {
	//	c.candleBuffer[market] = make(map[string]msg.Candle)
	//	// Close is either zero or fetch!!! when loading first time with historical data in database
	//	for _, interval := range supportedIntervals {
	//		bufferKey := getBufferKey(roundedTimestamps[interval], interval)
	//		c.candleBuffer[market][bufferKey] = *NewCandle(roundedTimestamps[interval], 0, 0, interval)
	//	}
	//}
}

func (c *candleStore) resetCandleBuffer(market string) {
	c.candleBuffer[market] = make(map[string]msg.Candle)
}

func getBufferKey(timestamp uint64, interval msg.Interval) string {
	return fmt.Sprintf("%d:%s", timestamp, interval.String())
}

func (c *candleStore) AddTradeToBuffer(market string, trade msg.Trade) error {
	//c.mu.Lock()
	//if _, ok := c.candleBuffer[market]; !ok {
	//	c.resetCandleBuffer(market, trade.Timestamp)
	//	fmt.Printf("Reseted\n")
	//	c.printCandleBuffer()
	//}
	//c.mu.Unlock()

	//fmt.Printf("AddTradeToBuffer called \n")
	// update all intervals for this market with trade data

	for _, interval := range supportedIntervals {
		roundedTradeTimestamp := vegatime.Stamp(trade.Timestamp).RoundToNearest(interval).UnixNano()
		bufferKey := getBufferKey(roundedTradeTimestamp, interval)
		if candle, exists := c.candleBuffer[market][bufferKey]; exists {
			UpdateCandle(&candle, &trade)
			c.candleBuffer[market][bufferKey] = candle
		} else {
			c.candleBuffer[market][bufferKey] = *NewCandle(roundedTradeTimestamp, trade.Price, trade.Size, candle.Interval)
		}
	}

	//fmt.Printf("\n\nc.candleBuffer[market]\n\n")
	//c.printCandleBuffer()

	return nil
}

func (c *candleStore) printCandleBuffer() {
	for market, val := range c.candleBuffer {
		fmt.Printf("Market = %s\n", market)
		for bufferKey, candle := range val {
			fmt.Printf("BK=%s	T=%d	I=%+v	V=%d	H=%d	C=%d\n", bufferKey, candle.Timestamp, candle.Interval, candle.Volume, candle.High, candle.Low)
		}
	}
}

func (c *candleStore) Close() {
	defer c.badger.db.Close()
}

func (c *candleStore) GenerateCandlesFromBuffer(market string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	//fmt.Printf("\nGenerateCandlesFromBuffer called \n\n")
	//c.printCandleBuffer()
	items := c.candleBuffer[market]
	c.insertOrUpdateCandle(market, items)
	//c.resetCandleBuffer(market)
	//fmt.Printf("\nEnd of GenerateCandlesFromBuffer \n\n")
	return nil
}

//func (c *candleStore) GenerateEmptyCandles(market string, timestamp uint64) error {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//	items := c.candleBuffer[market]
//	c.resetCandleBuffer(market)
//	c.insertOrUpdateCandle(market, items)
//	return nil
//}

func (c *candleStore) insertOrUpdateCandle(market string, items map[string]msg.Candle) error {

	wb := c.badger.db.NewWriteBatch()
	defer wb.Cancel()

	txn := c.badger.db.NewTransaction(true)
	for _, candle := range items {
		badgerKey := c.badger.candleKey(market, candle.Interval, candle.Timestamp)
		// check if key in db
		item, err := txn.Get(badgerKey)

		// if not present insert
		if err == badger.ErrKeyNotFound {
			candleBuf, err := proto.Marshal(&candle)
			if err != nil {
				return err
			}
			if err = wb.Set(badgerKey, candleBuf, 0); err != nil {
				return err
			}

			// notify candle
			log.Debugf("New Candle inserted %+v at %s \n", candle, string(badgerKey))
			c.QueueEvent(candle, candle.Interval)
			//fmt.Printf("Inserted and queued\n")
		}
		// if present run update and candle volume is greater than zero (there is something to update)
		if err == nil && candle.Volume != uint64(0) {
			// unmarshal fetched candle
			var candleFromDb msg.Candle
			itemCopy, _ := item.ValueCopy(nil)
			proto.Unmarshal(itemCopy, &candleFromDb)

			// update fetched candle with new trade
			MergeCandles(&candleFromDb, candle)

			// marshal candle
			candleBuf, err := proto.Marshal(&candleFromDb)
			if err != nil {
				return err
			}

			// push candle to badger
			if err = wb.Set(badgerKey, candleBuf, 0); err != nil {
				return err
			}

			// notify candle
			log.Debugf("Candle fetched, updated and inserted %+v at \n", candleFromDb, string(badgerKey))
			c.QueueEvent(candleFromDb, candleFromDb.Interval)
		}
	}

	//fmt.Printf("Done\n")
	if err := wb.Flush(); err != nil {
		wb.Cancel()
		return err
	}
	//fmt.Printf("OK\n")
	return nil
}

//func (c *candleStore) GenerateCandles2(trade *msg.Trade) error {
//
//	//given trade generate appropriate timestamps and badger keys for each interval
//	badgerKeys, candleTimestamps := c.generateKeysForTimestamp(trade.Market, trade.Timestamp)
//
//	// for each trade generate candle keys and run update on each record
//	txn := c.badger.db.NewTransaction(true)
//	for interval, badgerKey := range badgerKeys {
//
//		item, err := txn.Get(badgerKey)
//
//		// if key does not exist, insert candle for this timestamp
//		if err == badger.ErrKeyNotFound {
//			candle := NewCandle(uint64(candleTimestamps[interval]), trade.Price, trade.Size, interval)
//			candleBuf, err := proto.Marshal(candle)
//			if err != nil {
//				return err
//			}
//
//			if err = txn.Set(badgerKey, candleBuf); err != nil {
//				return err
//			}
//
//			log.Debugf("New Candle inserted %+v at \n", candle, string(badgerKey))
//
//			c.QueueEvent(*candle, interval)
//		}
//
//		// if key exists, update candle with this trade
//		if err == nil {
//
//			// unmarshal fetched candle
//			var candleForUpdate msg.Candle
//			itemCopy, err := item.ValueCopy(nil)
//			proto.Unmarshal(itemCopy, &candleForUpdate)
//
//			// update fetched candle with new trade
//			UpdateCandle(&candleForUpdate, trade)
//
//			// marshal candle
//			candleBuf, err := proto.Marshal(&candleForUpdate)
//			if err != nil {
//				return err
//			}
//
//			// push candle to badger
//			if err = txn.Set(badgerKey, candleBuf); err != nil {
//				return err
//			}
//
//			log.Debugf("Candle fetched, updated and inserted %+v at \n", candleForUpdate, string(badgerKey))
//
//			c.QueueEvent(candleForUpdate, interval)
//		}
//	}
//
//	if err := txn.Commit(); err != nil {
//		return err
//	}
//	return nil
//}
//
//func (c *candleStore) GenerateEmptyCandles(market string, timestamp uint64) error {
//
//	// flag to track if any new candle was generated used to notify observers of candle store
//	var generated bool
//
//	// generate keys for this timestamp
//	candleKeys, candleTimestamp := c.generateKeysForTimestamp(market, timestamp)
//
//	// if key does not exist seek most recent values, create empty candle with those close value and insert
//	txn := c.badger.db.NewTransaction(true)
//
//	// for all candle intervals
//	for interval, key := range candleKeys {
//
//		// if key does not exist, seek most recent value
//		_, err := txn.Get(key)
//		if err == badger.ErrKeyNotFound {
//
//			// find most recent candle
//			prefixForMostRecent, _ := c.badger.candlePrefix(market, interval, true)
//			previousCandle, err := c.fetchMostRecentCandle(txn, prefixForMostRecent)
//			if err != nil {
//				return err
//			}
//
//			// generate new candle based on the extracted close price
//			candleTimestamp := candleTimestamp[interval]
//			newCandle := NewCandle(uint64(candleTimestamp), previousCandle.Close, 0, interval)
//			candleBuf, err := proto.Marshal(newCandle)
//			if err != nil {
//				return err
//			}
//
//			// push new candle to badger
//			if err := txn.Set(key, candleBuf); err != nil {
//				return err
//			}
//
//			// push new candle onto the gql buffer for updates
//			c.QueueEvent(*newCandle, interval)
//
//			generated = true
//		}
//	}
//
//	if err := txn.Commit(); err != nil {
//		return err
//	}
//
//	// if any of the candles were updated push stacked changes and update the suppliers
//	if generated {
//		c.Notify()
//	}
//
//	return nil
//}
//
//func (c *candleStore) fetchMostRecentCandle(txn *badger.Txn, prefixForMostRecent []byte) (*msg.Candle, error) {
//	var previousCandle msg.Candle
//
//	// set iterator to reverse in order to fetch most recent
//	options := badger.DefaultIteratorOptions
//	options.Reverse = true
//
//	it := txn.NewIterator(options)
//	it.Seek(prefixForMostRecent)
//	defer it.Close()
//
//	item := it.Item()
//
//	value, err := item.ValueCopy(nil)
//	if err != nil {
//		return nil, err
//	}
//
//	proto.Unmarshal(value, &previousCandle)
//
//	return &previousCandle, nil
//}

//func (c *candleStore) generateKeysForTimestamp(market string, timestamp uint64) (map[msg.Interval][]byte, map[msg.Interval]uint64) {
//	keys := make(map[msg.Interval][]byte)
//	roundedTimestamps := getMapOfIntervalsToRoundedTimestamps(timestamp)
//
//	for interval, roundedTimestamp := range roundedTimestamps {
//		keys[interval] = c.badger.candleKey(market, interval, roundedTimestamp)
//	}
//
//	return keys, roundedTimestamps
//}

func (c *candleStore) fetchMostRecentCandle(txn *badger.Txn, prefixForMostRecent []byte) (*msg.Candle, error) {
	var previousCandle msg.Candle

	// set iterator to reverse in order to fetch most recent
	options := badger.DefaultIteratorOptions
	options.Reverse = true

	it := txn.NewIterator(options)
	it.Seek(prefixForMostRecent)
	defer it.Close()

	if !it.Valid() {
		return nil, errors.New("no candles for this market")
	}
	item := it.Item()

	value, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	proto.Unmarshal(value, &previousCandle)

	return &previousCandle, nil
}


func getMapOfIntervalsToRoundedTimestamps(timestamp uint64) map[msg.Interval]uint64 {
	// round timetamp to nearest minute, 5minute, 15 minute, hour, 6hours, 1 day intervals and return a map of rounded timestamps
	timestamps := make(map[msg.Interval]uint64)

	// round floor by integer division
	for _, interval := range supportedIntervals {
		timestamps[interval] = vegatime.Stamp(timestamp).RoundToNearest(interval).UnixNano()
	}

	return timestamps
}

func (c *candleStore) GetCandles(market string, sinceTimestamp uint64, interval msg.Interval) []*msg.Candle {

	// generate fetch key for the candles
	fetchKey := c.generateFetchKey(market, interval, sinceTimestamp)
	prefix, _ := c.badger.candlePrefix(market, interval, false)
	it := c.badger.getIterator(c.badger.db.NewTransaction(false), false)
	defer it.Close()

	var candles []*msg.Candle
	for it.Seek(fetchKey); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		value, err := item.ValueCopy(nil)
		if err != nil {
			fmt.Printf(err.Error())
			continue
		}

		var newCandle msg.Candle
		if err := proto.Unmarshal(value, &newCandle); err != nil {
			fmt.Printf(err.Error())
			continue
		}
		candles = append(candles, &newCandle)
	}

	return candles
}

func (c *candleStore) generateFetchKey(market string, interval msg.Interval, sinceTimestamp uint64) []byte {

	// returns valid key for market, interval and timestamp
	// round floor by integer division

	switch interval {
		case msg.Interval_I1M:
			timestampRoundedToMinute := vegatime.Stamp(sinceTimestamp).RoundToNearest(interval).UnixNano()
			return c.badger.candleKey(market, interval, timestampRoundedToMinute)
		case msg.Interval_I5M:
			timestampRoundedTo5Minutes := vegatime.Stamp(sinceTimestamp).RoundToNearest(interval).UnixNano()
			return c.badger.candleKey(market, interval, timestampRoundedTo5Minutes)
		case msg.Interval_I15M:
			timestampRoundedTo15Minutes := vegatime.Stamp(sinceTimestamp).RoundToNearest(interval).UnixNano()
			return c.badger.candleKey(market, interval, timestampRoundedTo15Minutes)
		case msg.Interval_I1H:
			timestampRoundedTo1Hour := vegatime.Stamp(sinceTimestamp).RoundToNearest(interval).UnixNano()
			return c.badger.candleKey(market, interval, timestampRoundedTo1Hour)
		case msg.Interval_I6H:
			timestampRoundedTo6Hour := vegatime.Stamp(sinceTimestamp).RoundToNearest(interval).UnixNano()
			return c.badger.candleKey(market, interval, timestampRoundedTo6Hour)
		case msg.Interval_I1D:
			timestampRoundedToDay := vegatime.Stamp(sinceTimestamp).RoundToNearest(interval).UnixNano()
			return c.badger.candleKey(market, interval, timestampRoundedToDay)
	}
	return nil
}

func NewCandle(timestamp, openPrice, size uint64, interval msg.Interval) *msg.Candle {
	//TODO: get candle form pool of candles
	datetime := vegatime.Stamp(timestamp).Rfc3339()
	return &msg.Candle{Timestamp: timestamp, Datetime: datetime, Open: openPrice, Close: openPrice,
		Low: openPrice, High: openPrice, Volume: size, Interval: interval}
}

func UpdateCandle(candle *msg.Candle, trade *msg.Trade) {
	// always overwrite close price
	candle.Close = trade.Price

	if candle.Open == uint64(0) {
		candle.Open = trade.Price
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

func MergeCandles(candleFromDB *msg.Candle, candleUpdate msg.Candle) {
	// always overwrite close price
	candleFromDB.Close = candleUpdate.Close

	// set minimum
	if candleUpdate.Low < candleFromDB.Low {
		candleFromDB.Low = candleUpdate.Low
	}

	// set maximum
	if candleUpdate.High > candleFromDB.High {
		candleFromDB.High = candleUpdate.High
	}

	candleFromDB.Volume += candleUpdate.Volume
}