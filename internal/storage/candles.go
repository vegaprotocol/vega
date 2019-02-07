package storage

import (
	"fmt"
	"sync"

	"vega/internal/vegatime"
	types "vega/proto"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

// CandleStore provides a set of functions that can manipulate a candle store, it provides a way for
// developers to create implementations of a CandleStore e.g. a RAM store or persistent store (badger)
type CandleStore interface {
	// Subscribe to a channel of new or updated candles. The subscriber id will be returned as a uint64 value
	// and must be retained for future reference and to unsubscribe.
	Subscribe(iT *InternalTransport) uint64

	// Unsubscribe from a candles channel. Provide the subscriber id you wish to stop receiving new events for.
	Unsubscribe(id uint64) error

	// StartNewBuffer creates a new trades buffer for the given market at timestamp.
	StartNewBuffer(market string, timestamp uint64) error

	// AddTradeToBuffer adds a trade to the trades buffer for the given market.
	AddTradeToBuffer(market string, trade types.Trade) error

	// GenerateCandlesFromBuffer will generate all candles for a given market.
	GenerateCandlesFromBuffer(market string) error

	// GetCandles returns all candles at interval since timestamp for a market.
	GetCandles(market string, sinceTimestamp uint64, interval types.Interval) ([]*types.Candle, error)

	// Close can be called to clean up and close any storage
	// connections held by the underlying storage mechanism.
	Close() error
}

// Currently we support 6 interval durations for trading candles on VEGA, as follows:
var supportedIntervals = [6]types.Interval{
	types.Interval_I1M,  // 1 minute
	types.Interval_I5M,  // 5 minutes
	types.Interval_I15M, // 15 minutes
	types.Interval_I1H,  // 1 hour
	types.Interval_I6H,  // 6 hours
	types.Interval_I1D,  // 1 day

	// Add intervals here as required...
}

// Monday, January 1, 2018 12:00:01 AM GMT+00:00
const minSinceTimestamp uint64 = 1514764801000

// badgerCandleStore is a package internal data struct that implements the CandleStore interface.
type badgerCandleStore struct {
	*Config
	badger       *badgerStore
	candleBuffer map[string]map[string]types.Candle
	subscribers  map[uint64]*InternalTransport
	subscriberId uint64
	queue        []marketCandle
	mu           sync.Mutex
}

// InternalTransport provides a data structure that holds an internal channel for a market and interval.
type InternalTransport struct {
	Market    string
	Interval  types.Interval
	Transport chan types.Candle
}

type marketCandle struct {
	Market string
	Candle types.Candle
}

// NewCandleStore is used to initialise and create a CandleStore, this implementation is currently
// using the badger k-v persistent storage engine under the hood. The caller will specify a dir to
// use as the storage location on disk for any stored files via Config.
func NewCandleStore(c *Config) (CandleStore, error) {
	db, err := badger.Open(customBadgerOptions(c.CandleStoreDirPath))
	if err != nil {
		return nil, errors.Wrap(err, "error opening badger database for candles storage")
	}
	bs := badgerStore{db: db}
	return &badgerCandleStore{
		Config:       c,
		badger:       &bs,
		subscribers:  make(map[uint64]*InternalTransport),
		candleBuffer: make(map[string]map[string]types.Candle),
		queue:        make([]marketCandle, 0),
	}, nil
}

// Subscribe to a channel of new or updated candles. The subscriber id will be returned as a uint64 value
// and must be retained for future reference and to unsubscribe.
func (c *badgerCandleStore) Subscribe(iT *InternalTransport) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.subscriberId = c.subscriberId + 1
	c.subscribers[c.subscriberId] = iT

	c.log.Debugf("CandleStore -> Subscribe: Candle subscriber added: %d", c.subscriberId)
	return c.subscriberId
}

// Unsubscribe from a candles channel. Provide the subscriber id you wish to stop receiving new events for.
func (c *badgerCandleStore) Unsubscribe(id uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.subscribers) == 0 {
		c.log.Debugf("CandleStore -> Unsubscribe: No subscribers connected")
		return nil
	}

	if _, exists := c.subscribers[id]; exists {
		delete(c.subscribers, id)
		c.log.Debugf("CandleStore -> Unsubscribe: Subscriber removed: %v", id)
		return nil
	}
	return errors.New(fmt.Sprintf("CandleStore subscriber does not exist with id: %d", id))
}

// Close can be called to clean up and close any storage
// connections held by the underlying storage mechanism.
func (c *badgerCandleStore) Close() error {
	return c.badger.db.Close()
}

// StartNewBuffer creates a new trades buffer for the given market at timestamp.
func (c *badgerCandleStore) StartNewBuffer(market string, timestamp uint64) error {
	roundedTimestamps := getMapOfIntervalsToRoundedTimestamps(timestamp)
	previousCandleBuffer := c.candleBuffer[market]
	c.resetCandleBuffer(market)

	for _, interval := range supportedIntervals {
		bufferKey := getBufferKey(roundedTimestamps[interval], interval)
		lastClose := previousCandleBuffer[bufferKey].Close
		if lastClose == uint64(0) {
			prefixForMostRecent, _ := c.badger.candlePrefix(market, interval, true)
			txn := c.badger.readTransaction()
			previousCandle, err := c.fetchMostRecentCandle(txn, prefixForMostRecent)
			if err != nil {
				lastClose = 0
			} else {
				lastClose = previousCandle.Close
			}
			txn.Discard()
		}
		c.candleBuffer[market][bufferKey] = *newCandle(roundedTimestamps[interval], lastClose, 0, interval)
	}

	return nil
}

// AddTradeToBuffer adds a trade to the trades buffer for the given market.
func (c *badgerCandleStore) AddTradeToBuffer(market string, trade types.Trade) error {

	for _, interval := range supportedIntervals {
		roundedTradeTimestamp := vegatime.Stamp(trade.Timestamp).RoundToNearest(interval).Uint64()
		bufferKey := getBufferKey(roundedTradeTimestamp, interval)

		// check if bufferKey is present in buffer
		if candle, exists := c.candleBuffer[market][bufferKey]; exists {
			// if exists update the value of the candle under bufferKey with trade data
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

// GenerateCandlesFromBuffer will generate all candles for a given market.
func (c *badgerCandleStore) GenerateCandlesFromBuffer(market string) error {

	fetchCandle := func(txn *badger.Txn, badgerKey []byte) (*types.Candle, error) {
		item, err := txn.Get(badgerKey)
		if err != nil {
			return nil, err
		}
		// unmarshal fetched candle
		var candleFromDb types.Candle
		itemCopy, _ := item.ValueCopy(nil)
		err = proto.Unmarshal(itemCopy, &candleFromDb)
		if err != nil {
			return nil, errors.Wrap(err, "fetchCandle unmarshal failed")
		}
		return &candleFromDb, nil
	}

	insertNewCandle := func(wb *badger.WriteBatch, badgerKey []byte, candle types.Candle) error {
		candleBuf, err := proto.Marshal(&candle)
		if err != nil {
			return err
		}
		if err = wb.Set(badgerKey, candleBuf, 0); err != nil {
			return err
		}
		return nil
	}

	updateCandle := func(wb *badger.WriteBatch, badgerKey []byte, candleDb *types.Candle) error {
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

	readTxn := c.badger.readTransaction()
	defer readTxn.Discard()

	writeBatch := c.badger.db.NewWriteBatch()
	defer writeBatch.Cancel()

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, candle := range c.candleBuffer[market] {
		badgerKey := c.badger.candleKey(market, candle.Interval, candle.Timestamp)
		candleDb, err := fetchCandle(readTxn, badgerKey)
		if err == badger.ErrKeyNotFound {
			insertNewCandle(writeBatch, badgerKey, candle)

			c.log.Debugf("new candle inserted %+v at %s", candle, string(badgerKey))
			c.queueEvent(market, candle)
		}
		if err == nil && candle.Volume != uint64(0) {
			// update fetched candle with new trade
			mergeCandles(candleDb, candle)
			updateCandle(writeBatch, badgerKey, candleDb)

			c.log.Debugf("candle updated %+v at %s", candleDb, string(badgerKey))
			c.queueEvent(market, *candleDb)
		}
	}

	if err := writeBatch.Flush(); err != nil {
		return err
	}

	// now push new updates to any observers
	err := c.notify()
	if err != nil {
		return err
	}

	return nil
}

// GetCandles returns all candles at interval since timestamp for a market.
func (c *badgerCandleStore) GetCandles(market string, sinceTimestamp uint64, interval types.Interval) ([]*types.Candle, error) {
	if sinceTimestamp < minSinceTimestamp {
		return nil, errors.New("invalid sinceTimestamp, ensure format is epoch+nanoseconds timestamp")
	}

	// generate fetch key for the candles
	fetchKey := c.generateFetchKey(market, interval, sinceTimestamp)
	prefix, _ := c.badger.candlePrefix(market, interval, false)

	txn := c.badger.readTransaction()
	defer txn.Discard()

	it := c.badger.getIterator(txn, false)
	defer it.Close()

	var candles []*types.Candle
	for it.Seek(fetchKey); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		value, err := item.ValueCopy(nil)
		if err != nil {
			c.log.Errorf("error getting candle value: %s", err)
			continue
		}

		var newCandle types.Candle
		if err := proto.Unmarshal(value, &newCandle); err != nil {
			c.log.Errorf("unmarshal failed %s", err.Error())
			continue
		}
		candles = append(candles, &newCandle)
	}

	return candles, nil
}

// generateFetchKey calculates the correct badger key for the given market, interval and timestamp.
func (c *badgerCandleStore) generateFetchKey(market string, interval types.Interval, sinceTimestamp uint64) []byte {
	// returns valid key for Market, interval and timestamp
	// round floor by integer division
	switch interval {
	case types.Interval_I1M:
		timestampRoundedToMinute := vegatime.Stamp(sinceTimestamp).RoundToNearest(interval).Uint64()
		return c.badger.candleKey(market, interval, timestampRoundedToMinute)
	case types.Interval_I5M:
		timestampRoundedTo5Minutes := vegatime.Stamp(sinceTimestamp).RoundToNearest(interval).Uint64()
		return c.badger.candleKey(market, interval, timestampRoundedTo5Minutes)
	case types.Interval_I15M:
		timestampRoundedTo15Minutes := vegatime.Stamp(sinceTimestamp).RoundToNearest(interval).Uint64()
		return c.badger.candleKey(market, interval, timestampRoundedTo15Minutes)
	case types.Interval_I1H:
		timestampRoundedTo1Hour := vegatime.Stamp(sinceTimestamp).RoundToNearest(interval).Uint64()
		return c.badger.candleKey(market, interval, timestampRoundedTo1Hour)
	case types.Interval_I6H:
		timestampRoundedTo6Hour := vegatime.Stamp(sinceTimestamp).RoundToNearest(interval).Uint64()
		return c.badger.candleKey(market, interval, timestampRoundedTo6Hour)
	case types.Interval_I1D:
		timestampRoundedToDay := vegatime.Stamp(sinceTimestamp).RoundToNearest(interval).Uint64()
		return c.badger.candleKey(market, interval, timestampRoundedToDay)
	}
	return nil
}

func (c *badgerCandleStore) fetchMostRecentCandle(txn *badger.Txn, prefixForMostRecent []byte) (*types.Candle, error) {
	var previousCandle types.Candle

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

	err = proto.Unmarshal(value, &previousCandle)
	if err != nil {
		return nil, errors.Wrap(err, "previous candle unmarshal failed")
	}

	return &previousCandle, nil
}

// getMapOfIntervalsToRoundedTimestamps rounds timestamp to nearest minute, 5minute,
//  15 minute, hour, 6hours, 1 day intervals and return a map of rounded timestamps
func getMapOfIntervalsToRoundedTimestamps(timestamp uint64) map[types.Interval]uint64 {
	timestamps := make(map[types.Interval]uint64)

	// round floor by integer division
	for _, interval := range supportedIntervals {
		timestamps[interval] = vegatime.Stamp(timestamp).RoundToNearest(interval).Uint64()
	}

	return timestamps
}

// queueEvent appends a candle onto a queue for a market.
func (c *badgerCandleStore) queueEvent(market string, candle types.Candle) {
	c.queue = append(c.queue, marketCandle{Market: market, Candle: candle})
}

// notify sends out any candles in the buffer to subscribers. If there are no
// subscribers or the queue is empty it will return with no work.
func (c *badgerCandleStore) notify() error {
	if len(c.subscribers) == 0 {
		c.log.Debugf("CandleStore -> Notify: No subscribers connected")
		return nil
	}
	if len(c.queue) == 0 {
		c.log.Debugf("CandleStore -> Notify: No candles in the queue")
		return nil
	}

	c.log.Debugf("%d candles in the notify queue for %d subscribers", len(c.queue), len(c.subscribers))

	// update candle for each subscriber, only if there are candles in the queue
	for id, iT := range c.subscribers {

		c.log.Debugf("Candle subscriber %d (%s) ready to notify", id, iT.Market)

		for _, item := range c.queue {

			// find candle with right interval
			if item.Candle.Interval != iT.Interval {
				continue
			}

			c.log.Infof("Doing update for subscriber %d subscribing %s (%s)", id, iT.Interval, iT.Market)

			// try to place candle onto transport
			select {
			case iT.Transport <- item.Candle:
				c.log.Infof("Candle updated for subscriber %d at interval: %s (%s)", id, item.Candle.Interval, iT.Market)
			default:
				c.log.Infof("Candles state could not been updated for subscriber %d at interval %s (%s)", id, item.Candle.Interval, iT.Market)
			}
			break
		}

		c.log.Debugf("Candle subscriber %d (%s) notified for interval %s", id, iT.Market, iT.Interval)
	}

	c.queue = make([]marketCandle, 0)

	return nil
}

// resetCandleBuffer does what it says on the tin :)
func (c *badgerCandleStore) resetCandleBuffer(market string) {
	c.candleBuffer[market] = make(map[string]types.Candle)
}

// getBufferKey returns the custom formatted buffer key for internal trade to timestamp mapping.
func getBufferKey(timestamp uint64, interval types.Interval) string {
	return fmt.Sprintf("%d:%s", timestamp, interval.String())
}

// printCandleBuffer is used for debugging the output of the generation methods.
func (c *badgerCandleStore) printCandleBuffer() {
	for market, val := range c.candleBuffer {
		c.log.Debugf("Market: %s", market)
		for bufferKey, candle := range val {
			c.log.Debugf("BK=%s	T=%d	I=%+v	V=%d	H=%d	C=%d", bufferKey, candle.Timestamp, candle.Interval, candle.Volume, candle.High, candle.Low)
		}
	}
}

// newCandle constructs a new candle with minimum required parameters.
func newCandle(timestamp, openPrice, size uint64, interval types.Interval) *types.Candle {
	candle := types.CandlePool.Get().(*types.Candle)
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

// mergeCandles is used to update an existing candle in the buffer.
func mergeCandles(candleFromDB *types.Candle, candleUpdate types.Candle) {
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
