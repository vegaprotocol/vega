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

type CandleStore interface {
	Close()
	Subscribe(iT *InternalTransport) uint64
	Unsubscribe(id uint64) error
	Notify() error

	StartNewBuffer(market string, timestamp uint64)
	AddTradeToBuffer(market string, trade msg.Trade) error
	GenerateCandlesFromBuffer(market string) error

	GetCandles(market string, sinceTimestamp uint64, interval msg.Interval) []*msg.Candle
}

var supportedIntervals = [6]msg.Interval{
	msg.Interval_I1M, msg.Interval_I5M, msg.Interval_I15M, msg.Interval_I1H, msg.Interval_I6H, msg.Interval_I1D,}

type candleStore struct {
	badger *badgerStore

	NotifyQueue []QueueItem
	subscribers  map[uint64]*InternalTransport
	candleBuffer map[string]map[string]msg.Candle
	subscriberId uint64
	mu           sync.Mutex
}

type InternalTransport struct {
	Market    string
	Interval msg.Interval
	Transport chan msg.Candle
}

type QueueItem struct {
	Market string
	Candle msg.Candle
}

func NewCandleStore(dir string) CandleStore {
	db, err := badger.Open(customBadgerOptions(dir))
	if err != nil {
		fmt.Printf(err.Error())
	}
	bs := badgerStore{db: db}
	return &candleStore{badger: &bs,
		subscribers: make(map[uint64]*InternalTransport),
		candleBuffer: make(map[string]map[string]msg.Candle),
		NotifyQueue: make([]QueueItem, 0),
	}
}

func (c *candleStore) Subscribe(iT *InternalTransport) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.subscriberId = c.subscriberId + 1
	c.subscribers[c.subscriberId] = iT
	log.Debugf("CandleStore -> Subscribe: Candle subscriber added: %d", c.subscriberId)
	return c.subscriberId
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

func (c *candleStore) QueueEvent(market string, candle msg.Candle) {
	c.NotifyQueue = append(c.NotifyQueue, QueueItem{Market:market, Candle:candle})
}

func (c *candleStore) Notify() error {

	if len(c.subscribers) == 0 {
		log.Debugf("CandleStore -> Notify: No subscribers connected")
		return nil
	}

	// update candle for each subscriber
	for _, item := range c.NotifyQueue {
		log.Infof("Propagating %+v", item.Candle)
		for id, iT := range c.subscribers {
			log.Infof("Doing update for subscriber %d subscribing %s", id, item.Candle.Interval)
			// find candle with right interval
			if item.Candle.Interval != iT.Interval {
				continue
			}

			// try to place candle onto transport
			select {
			case iT.Transport <- item.Candle:
				log.Infof("Candle updated for interval: %s", item.Candle.Interval)
			default:
				log.Infof("Candles state could not been updated for subscriber %d at interval %s", id, item.Candle.Interval)
			}
			break
		}
	}

	c.NotifyQueue = make([]QueueItem, 0)

	return nil
}

func (c *candleStore) Close() {
	defer c.badger.db.Close()
}

func (c *candleStore) StartNewBuffer(market string, timestamp uint64) {
	roundedTimestamps := getMapOfIntervalsToRoundedTimestamps(timestamp)

	// keep previous state
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
		c.candleBuffer[market][bufferKey] = *newCandle(roundedTimestamps[interval], lastClose, 0, interval)
	}
}

func (c *candleStore) AddTradeToBuffer(market string, trade msg.Trade) error {

	for _, interval := range supportedIntervals {
		roundedTradeTimestamp := vegatime.Stamp(trade.Timestamp).RoundToNearest(interval).UnixNano()
		bufferKey := getBufferKey(roundedTradeTimestamp, interval)

		// check if bufferKey is present in buffer
		if candle, exists := c.candleBuffer[market][bufferKey]; exists {
			// if exists update the value of the canle under bufferKey with trade data
			updateCandle(&candle, &trade)
			c.candleBuffer[market][bufferKey] = candle
		} else {
			// if doesn't exist create new candle under this buffer key
			c.candleBuffer[market][bufferKey] = *newCandle(roundedTradeTimestamp, trade.Price, trade.Size, candle.Interval)
		}
	}

	//c.printCandleBuffer()
	return nil
}

func (c *candleStore) GenerateCandlesFromBuffer(market string) error {

	fetchCandle := func(txn *badger.Txn, badgerKey []byte) (*msg.Candle, error) {
		item, err := txn.Get(badgerKey)
		if err != nil {
			return nil, err
		}
		// unmarshal fetched candle
		var candleFromDb msg.Candle
		itemCopy, _ := item.ValueCopy(nil)
		proto.Unmarshal(itemCopy, &candleFromDb)
		return &candleFromDb, nil
	}

	insertNewCandle := func(wb *badger.WriteBatch, badgerKey []byte, candle msg.Candle) error {
		candleBuf, err := proto.Marshal(&candle)
		if err != nil {
			return err
		}
		if err = wb.Set(badgerKey, candleBuf, 0); err != nil {
			return err
		}
		return nil
	}

	updateCandle := func(wb *badger.WriteBatch, badgerKey []byte, candleDb *msg.Candle) error {
		// marshal candle
		candleBuf, err := proto.Marshal(candleDb)
		if err != nil {
			return err
		}

		// push candle to badger
		if err = wb.Set(badgerKey, candleBuf, 0); err != nil {
			return err
		}
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	readTxn := c.badger.db.NewTransaction(true)
	writeBatch := c.badger.db.NewWriteBatch()
	defer writeBatch.Cancel()
	for _, candle := range c.candleBuffer[market] {
		badgerKey := c.badger.candleKey(market, candle.Interval, candle.Timestamp)
		candleDb, err := fetchCandle(readTxn, badgerKey)
		if err == badger.ErrKeyNotFound {
			insertNewCandle(writeBatch, badgerKey, candle)
			log.Debugf("new Candle inserted %+v at %s \n", candle, string(badgerKey))
			c.QueueEvent(market, candle)
		}
		if err == nil && candle.Volume != uint64(0){
			// update fetched candle with new trade
			mergeCandles(candleDb, candle)
			updateCandle(writeBatch, badgerKey, candleDb)
			log.Debugf("candle updated %+v at \n", candleDb, string(badgerKey))
			c.QueueEvent(market, candle)
		}
	}

	if err := writeBatch.Flush(); err != nil {
		writeBatch.Cancel()
		return err
	}

	c.Notify()

	return nil
}

func (c *candleStore) resetCandleBuffer(market string) {
	c.candleBuffer[market] = make(map[string]msg.Candle)
}

func getBufferKey(timestamp uint64, interval msg.Interval) string {
	return fmt.Sprintf("%d:%s", timestamp, interval.String())
}

func (c *candleStore) printCandleBuffer() {
	for market, val := range c.candleBuffer {
		log.Debugf("Market = %s\n", market)
		for bufferKey, candle := range val {
			log.Debugf("BK=%s	T=%d	I=%+v	V=%d	H=%d	C=%d\n", bufferKey, candle.Timestamp, candle.Interval, candle.Volume, candle.High, candle.Low)
		}
	}
}

func (c *candleStore) fetchMostRecentCandle(txn *badger.Txn, prefixForMostRecent []byte) (*msg.Candle, error) {
	var previousCandle msg.Candle

	// set iterator to reverse in order to fetch most recent
	options := badger.DefaultIteratorOptions
	options.Reverse = true

	it := txn.NewIterator(options)
	it.Seek(prefixForMostRecent)
	defer it.Close()

	if !it.Valid() {
		return nil, errors.New("no candles for this Market")
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
	// returns valid key for Market, interval and timestamp
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

func newCandle(timestamp, openPrice, size uint64, interval msg.Interval) *msg.Candle {
	candle := msg.CandlePool.Get().(*msg.Candle)
	candle.Timestamp = timestamp
	candle.Datetime = vegatime.Stamp(timestamp).Rfc3339()
	candle.High = openPrice
	candle.Low = openPrice
	candle.Open = openPrice
	candle.Close = openPrice
	candle.Volume = size
	candle.Interval = interval
	return candle
}

func updateCandle(candle *msg.Candle, trade *msg.Trade) {
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

func mergeCandles(candleFromDB *msg.Candle, candleUpdate msg.Candle) {
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
