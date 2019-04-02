package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

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
const minSinceTimestamp int64 = 1514764801000

var minSinceTime time.Time = vegatime.UnixNano(minSinceTimestamp)

// Candle is a package internal data struct that implements the CandleStore interface.
type Candle struct {
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
	Transport chan *types.Candle
}

type marketCandle struct {
	Market string
	Candle types.Candle
}

// NewCandles is used to initialise and create a CandleStore, this implementation is currently
// using the badger k-v persistent storage engine under the hood. The caller will specify a dir to
// use as the storage location on disk for any stored files via Config.
func NewCandles(c *Config) (*Candle, error) {
	err := InitStoreDirectory(c.CandleStoreDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "error on init badger database for candles storage")
	}
	db, err := badger.Open(customBadgerOptions(c.CandleStoreDirPath, c.GetLogger()))
	if err != nil {
		return nil, errors.Wrap(err, "error opening badger database for candles storage")
	}
	bs := badgerStore{db: db}
	return &Candle{
		Config:       c,
		badger:       &bs,
		subscribers:  make(map[uint64]*InternalTransport),
		candleBuffer: make(map[string]map[string]types.Candle),
		queue:        make([]marketCandle, 0),
	}, nil
}

// Subscribe to a channel of new or updated candles. The subscriber id will be returned as a uint64 value
// and must be retained for future reference and to unsubscribe.
func (c *Candle) Subscribe(iT *InternalTransport) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.subscriberId = c.subscriberId + 1
	c.subscribers[c.subscriberId] = iT

	c.log.Debug("Candle subscriber added in candle store",
		logging.Uint64("subscriber-id", c.subscriberId))

	return c.subscriberId
}

// Unsubscribe from a candles channel. Provide the subscriber id you wish to stop receiving new events for.
func (c *Candle) Unsubscribe(id uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.subscribers) == 0 {
		c.log.Debug("Un-subscribe called in candle store, no subscribers connected",
			logging.Uint64("subscriber-id", id))

		return nil
	}

	if _, exists := c.subscribers[id]; exists {
		delete(c.subscribers, id)

		c.log.Debug("Un-subscribe called in candle store, subscriber removed",
			logging.Uint64("subscriber-id", id))

		return nil
	}

	c.log.Warn("Un-subscribe called in candle store, subscriber does not exist",
		logging.Uint64("subscriber-id", id))

	return errors.New(fmt.Sprintf("Candle store subscriber does not exist with id: %d", id))
}

// Close can be called to clean up and close any storage
// connections held by the underlying storage mechanism.
func (c *Candle) Close() error {
	return c.badger.db.Close()
}

// StartNewBuffer creates a new trades buffer for the given market at timestamp.
func (c *Candle) StartNewBuffer(marketId string, timestamp time.Time) error {
	roundedTimestamps := GetMapOfIntervalsToRoundedTimestamps(timestamp)
	previousCandleBuffer := c.candleBuffer[marketId]
	c.resetCandleBuffer(marketId)

	for _, interval := range supportedIntervals {
		bufferKey := getBufferKey(roundedTimestamps[interval], interval)
		lastClose := previousCandleBuffer[bufferKey].Close
		if lastClose == uint64(0) {
			prefixForMostRecent, _ := c.badger.candlePrefix(marketId, interval, true)
			txn := c.badger.readTransaction()
			previousCandle, err := c.fetchMostRecentCandle(txn, prefixForMostRecent)
			if err != nil {
				lastClose = 0
			} else {
				lastClose = previousCandle.Close
			}
			txn.Discard()
		}
		c.candleBuffer[marketId][bufferKey] = *newCandle(roundedTimestamps[interval], lastClose, 0, interval)
	}

	return nil
}

// AddTradeToBuffer adds a trade to the trades buffer for the given market.
func (c *Candle) AddTradeToBuffer(trade types.Trade) error {

	if c.candleBuffer[trade.Market] == nil {
		c.log.Info("Starting new candle buffer for market",
			logging.String("market-id", trade.Market),
			logging.Int64("timestamp", trade.Timestamp))

		err := c.StartNewBuffer(trade.Market, vegatime.UnixNano(trade.Timestamp))
		if err != nil {
			return errors.Wrap(err, "Failed to start new buffer when adding trade to candle store")
		}
	}

	for _, interval := range supportedIntervals {
		roundedTradeTimestamp := vegatime.RoundToNearest(vegatime.UnixNano(trade.Timestamp), interval)

		bufferKey := getBufferKey(roundedTradeTimestamp, interval)

		// check if bufferKey is present in buffer
		if candle, exists := c.candleBuffer[trade.Market][bufferKey]; exists {
			// if exists update the value of the candle under bufferKey with trade data
			updateCandle(&candle, &trade)
			c.candleBuffer[trade.Market][bufferKey] = candle
		} else {
			// if doesn't exist create new candle under this buffer key
			c.candleBuffer[trade.Market][bufferKey] = *newCandle(roundedTradeTimestamp, trade.Price, trade.Size, candle.Interval)
		}
	}

	return nil
}

// GenerateCandlesFromBuffer will generate all candles for a given market.
func (c *Candle) GenerateCandlesFromBuffer(marketId string) error {

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
			c.log.Error("Failed to unmarshal candle value from badger in candle store (fetchCandle)",
				logging.Error(err),
				logging.String("badger-key", string(item.Key())),
				logging.String("raw-bytes", string(itemCopy)))

			return nil, errors.Wrap(err, "failed to unmarshal from badger (fetchCandle)")
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
		candleBuf, err := proto.Marshal(candleDb)
		if err != nil {
			return err
		}
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

	for _, candle := range c.candleBuffer[marketId] {
		badgerKey := c.badger.candleKey(marketId, candle.Interval, candle.Timestamp)
		candleDb, err := fetchCandle(readTxn, badgerKey)
		if err == badger.ErrKeyNotFound {
			err := insertNewCandle(writeBatch, badgerKey, candle)
			if err != nil {
				c.log.Error("Failed to insert new candle in candle store",
					logging.Candle(candle),
					logging.Error(err))
			} else {
				c.log.Debug("New candle inserted in candle store",
					logging.Candle(candle),
					logging.String("badger-key", string(badgerKey)))
			}
			c.queueEvent(marketId, candle)
		}

		if err == nil && candle.Volume != uint64(0) {
			// update fetched candle with new trade
			mergeCandles(candleDb, candle)
			err = updateCandle(writeBatch, badgerKey, candleDb)
			if err != nil {
				c.log.Error("Failed to update candle in candle store",
					logging.Candle(candle),
					logging.CandleWithTag(*candleDb, "existing-candle"),
					logging.Error(err))
			} else {
				c.log.Debug("Candle updated in candle store",
					logging.Candle(candle),
					logging.CandleWithTag(*candleDb, "existing-candle"),
					logging.String("badger-key", string(badgerKey)))
			}

			c.queueEvent(marketId, *candleDb)
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
func (c *Candle) GetCandles(ctx context.Context, market string, sinceTimestamp time.Time, interval types.Interval) ([]*types.Candle, error) {
	if sinceTimestamp.Before(minSinceTime) {
		return nil, errors.New("invalid sinceTimestamp, ensure format is epoch+nanoseconds timestamp")
	}

	// generate fetch key for the candles
	fetchKey := c.generateFetchKey(market, interval, sinceTimestamp.UnixNano())
	prefix, _ := c.badger.candlePrefix(market, interval, false)

	txn := c.badger.readTransaction()
	defer txn.Discard()

	it := c.badger.getIterator(txn, false)
	defer it.Close()

	ctx, cancel := context.WithTimeout(ctx, c.Config.Timeout*time.Second)
	defer cancel()
	deadline, _ := ctx.Deadline()

	var candles []*types.Candle
	for it.Seek(fetchKey); it.ValidForPrefix(prefix); it.Next() {
		select {
		case <-ctx.Done():
			if deadline.Before(time.Now()) {
				return nil, ErrTimeoutReached
			}
			return nil, nil
		default:
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {
				c.log.Error("Failure loading candle value from candle store (GetCandles)",
					logging.String("badger-key", string(item.Key())),
					logging.Error(err))
				continue
			}

			var newCandle types.Candle
			if err := proto.Unmarshal(value, &newCandle); err != nil {
				c.log.Error("Failed to unmarshal candle value from badger in candle store (GetCandles)",
					logging.Error(err),
					logging.String("badger-key", string(item.Key())),
					logging.String("raw-bytes", string(value)))
				continue
			}
			candles = append(candles, &newCandle)
		}
	}

	return candles, nil
}

// generateFetchKey calculates the correct badger key for the given market, interval and timestamp.
func (c *Candle) generateFetchKey(market string, interval types.Interval, sinceTimestamp int64) []byte {
	// returns valid key for Market, interval and timestamp
	// round floor by integer division
	switch interval {
	case types.Interval_I1M:
		fallthrough
	case types.Interval_I5M:
		fallthrough
	case types.Interval_I15M:
		fallthrough
	case types.Interval_I1H:
		fallthrough
	case types.Interval_I6H:
		fallthrough
	case types.Interval_I1D:
		return c.badger.candleKey(market, interval,
			vegatime.RoundToNearest(vegatime.UnixNano(sinceTimestamp), interval).UnixNano())
	default:
		return nil
	}

}

func (c *Candle) fetchMostRecentCandle(txn *badger.Txn, prefixForMostRecent []byte) (*types.Candle, error) {
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

// GetMapOfIntervalsToRoundedTimestamps rounds timestamp to nearest minute, 5minute,
//  15 minute, hour, 6hours, 1 day intervals and return a map of rounded timestamps
func GetMapOfIntervalsToRoundedTimestamps(timestamp time.Time) map[types.Interval]time.Time {
	timestamps := make(map[types.Interval]time.Time)

	// round floor by integer division
	for _, interval := range supportedIntervals {
		timestamps[interval] = vegatime.RoundToNearest(timestamp, interval)
	}

	return timestamps
}

// queueEvent appends a candle onto a queue for a market.
func (c *Candle) queueEvent(market string, candle types.Candle) {
	c.queue = append(c.queue, marketCandle{Market: market, Candle: candle})
}

// notify sends out any candles in the buffer to subscribers. If there are no
// subscribers or the queue is empty it will return with no work.
func (c *Candle) notify() error {
	if len(c.subscribers) == 0 {
		c.log.Debug("No subscribers connected in candle store")
		return nil
	}
	if len(c.queue) == 0 {
		c.log.Debug("No candles queued in candle store")
		return nil
	}

	c.log.Debug("Candles in the subscription queue",
		logging.Int("queue-length", len(c.queue)),
		logging.Int("subscribers", len(c.subscribers)))

	// update candle for each subscriber, only if there are candles in the queue
	for id, iT := range c.subscribers {

		c.log.Debug("Candle subscriber ready to notify",
			logging.Uint64("id", id),
			logging.String("market", iT.Market))

		for _, item := range c.queue {
			item := item
			// find candle with right interval
			if item.Candle.Interval != iT.Interval {
				continue
			}

			c.log.Debug("About to update candle subscriber",
				logging.Uint64("id", id),
				logging.String("interval", iT.Interval.String()),
				logging.String("market", iT.Market))

			// try to place candle onto transport
			select {
			case iT.Transport <- &item.Candle:
				c.log.Debug("Candle updated for subscriber successfully",
					logging.Uint64("id", id),
					logging.String("interval", item.Candle.Interval.String()),
					logging.String("market", item.Market))
			default:
				c.log.Debug("Candle could not be updated for subscriber",
					logging.Uint64("id", id),
					logging.String("interval", item.Candle.Interval.String()),
					logging.String("market", item.Market))
			}
			break
		}

		c.log.Debug("Candle subscriber notified",
			logging.Uint64("id", id),
			logging.String("interval", iT.Interval.String()),
			logging.String("market", iT.Market))
	}

	c.queue = make([]marketCandle, 0)

	return nil
}

// resetCandleBuffer does what it says on the tin :)
func (c *Candle) resetCandleBuffer(market string) {
	c.candleBuffer[market] = make(map[string]types.Candle)
}

// getBufferKey returns the custom formatted buffer key for internal trade to timestamp mapping.
func getBufferKey(timestamp time.Time, interval types.Interval) string {
	return fmt.Sprintf("%d:%s", timestamp.UnixNano(), interval.String())
}

// newCandle constructs a new candle with minimum required parameters.
func newCandle(timestamp time.Time, openPrice, size uint64, interval types.Interval) *types.Candle {
	candle := types.CandlePool.Get().(*types.Candle)
	candle.Timestamp = timestamp.UnixNano()
	candle.Datetime = vegatime.Format(timestamp)
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
