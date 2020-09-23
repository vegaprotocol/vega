package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/dgraph-io/badger/v2"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// Monday, January 1, 2018 12:00:01 AM GMT+00:00
const minSinceTimestamp int64 = 1514764801000

var minSinceTime = vegatime.UnixNano(minSinceTimestamp)

// Candle is a package internal data struct that implements the CandleStore interface.
type Candle struct {
	Config

	cfgMu           sync.Mutex
	log             *logging.Logger
	badger          *badgerStore
	subscribers     map[uint64]*InternalTransport
	subscriberID    uint64
	queue           []marketCandle
	mu              sync.Mutex
	onCriticalError func()
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
func NewCandles(log *logging.Logger, c Config, onCriticalError func()) (*Candle, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	err := InitStoreDirectory(c.CandlesDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "error on init badger database for candles storage")
	}
	db, err := badger.Open(getOptionsFromConfig(c.Candles, c.CandlesDirPath, log))
	if err != nil {
		return nil, errors.Wrap(err, "error opening badger database for candles storage")
	}
	bs := badgerStore{db: db}
	return &Candle{
		log:             log,
		Config:          c,
		badger:          &bs,
		subscribers:     make(map[uint64]*InternalTransport),
		queue:           make([]marketCandle, 0),
		onCriticalError: onCriticalError,
	}, nil
}

// ReloadConf update the internal Candle configuration
func (c *Candle) ReloadConf(cfg Config) {
	c.log.Info("reloading configuration")
	if c.log.GetLevel() != cfg.Level.Get() {
		c.log.Info("updating log level",
			logging.String("old", c.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		c.log.SetLevel(cfg.Level.Get())
	}

	// only Timeout is really use in here
	c.cfgMu.Lock()
	c.Config = cfg
	c.cfgMu.Unlock()
}

// Subscribe to a channel of new or updated candles. The subscriber id will be returned as a uint64 value
// and must be retained for future reference and to unsubscribe.
func (c *Candle) Subscribe(iT *InternalTransport) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.subscriberID++
	c.subscribers[c.subscriberID] = iT

	c.log.Debug("Candle subscriber added in candle store",
		logging.Uint64("subscriber-id", c.subscriberID))

	return c.subscriberID
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

	return fmt.Errorf("subscriber to Candle store does not exist with id: %d", id)
}

// Close can be called to clean up and close any storage
// connections held by the underlying storage mechanism.
func (c *Candle) Close() error {
	return c.badger.db.Close()
}

// GenerateCandlesFromBuffer will generate all candles for a given market.
func (c *Candle) GenerateCandlesFromBuffer(marketID string, buf map[string]types.Candle) error {

	fetchCandle := func(txn *badger.Txn, badgerKey []byte) (*types.Candle, error) {
		item, err := txn.Get(badgerKey)
		if err != nil {
			return nil, err
		}
		// unmarshal fetched candle
		var candleFromDB types.Candle
		itemCopy, _ := item.ValueCopy(nil)
		err = proto.Unmarshal(itemCopy, &candleFromDB)
		if err != nil {
			c.log.Error("Failed to unmarshal candle value from badger in candle store (fetchCandle)",
				logging.Error(err),
				logging.String("badger-key", string(item.Key())))

			return nil, errors.Wrap(err, "failed to unmarshal from badger (fetchCandle)")
		}
		return &candleFromDB, nil
	}

	insertNewCandle := func(wb *badger.WriteBatch, badgerKey []byte, candle types.Candle) error {
		candleBuf, err := proto.Marshal(&candle)
		if err != nil {
			return err
		}
		if err = wb.Set(badgerKey, candleBuf); err != nil {
			return err
		}
		return nil
	}

	updateLastCandle := func(wb *badger.WriteBatch, key []byte, candleKey []byte) error {
		if err := wb.Set(key, candleKey); err != nil {
			return err
		}
		return nil
	}

	updateCandle := func(wb *badger.WriteBatch, badgerKey []byte, candleDB *types.Candle) error {
		candleBuf, err := proto.Marshal(candleDB)
		if err != nil {
			return err
		}
		if err = wb.Set(badgerKey, candleBuf); err != nil {
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

	for _, candle := range buf {
		badgerKey := c.badger.candleKey(marketID, candle.Interval, candle.Timestamp)
		candleDB, err := fetchCandle(readTxn, badgerKey)
		if err == badger.ErrKeyNotFound {
			// Do not overwrite err var, it is used below.
			subErr := insertNewCandle(writeBatch, badgerKey, candle)
			if subErr != nil {
				c.log.Error("Failed to insert new candle in candle store",
					logging.Candle(candle),
					logging.Error(subErr))
				c.onCriticalError()
			} else {
				if c.log.GetLevel() == logging.DebugLevel {
					c.log.Debug("New candle inserted in candle store",
						logging.Candle(candle),
						logging.String("badger-key", string(badgerKey)))
				}
			}
			c.queueEvent(marketID, candle)
		}

		if err == nil && candle.Volume != uint64(0) {
			// update fetched candle with new trade
			mergeCandles(candleDB, candle)
			err = updateCandle(writeBatch, badgerKey, candleDB)
			if err != nil {
				c.log.Error("Failed to update candle in candle store",
					logging.Candle(candle),
					logging.CandleWithTag(*candleDB, "existing-candle"),
					logging.Error(err))
				c.onCriticalError()
			} else {
				if c.log.GetLevel() == logging.DebugLevel {
					c.log.Debug("Candle updated in candle store",
						logging.Candle(candle),
						logging.CandleWithTag(*candleDB, "existing-candle"),
						logging.String("badger-key", string(badgerKey)))
				}
			}

			c.queueEvent(marketID, *candleDB)
		}

		// add the lastCandle index
		lastCandleKey := c.badger.lastCandleKey(marketID, candle.Interval)
		err = updateLastCandle(writeBatch, lastCandleKey, badgerKey)
		if err != nil {
			c.log.Error("failed to store last candle",
				logging.Error(err),
				logging.String("market-id", marketID),
				logging.String("interval", candle.Interval.String()),
			)
			c.onCriticalError()
		} else {
			c.log.Debug("last candle updated",
				logging.String("market-id", marketID),
				logging.String("interval", candle.Interval.String()),
			)
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
func (c *Candle) GetCandles(ctx context.Context, market string, since time.Time, interval types.Interval) ([]*types.Candle, error) {
	if since.Before(minSinceTime) {
		return nil, errors.New("invalid sinceTimestamp, ensure format is epoch+nanoseconds timestamp")
	}

	// generate fetch key for the candles
	fetchKey := c.generateFetchKey(market, interval, since)
	prefix, _ := c.badger.candlePrefix(market, interval, false)

	ctx, cancel := context.WithTimeout(ctx, c.Config.Timeout.Duration)
	defer cancel()
	deadline, _ := ctx.Deadline()

	txn := c.badger.readTransaction()
	defer txn.Discard()

	it := c.badger.getIterator(txn, false)
	defer it.Close()

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
					logging.String("badger-key", string(item.Key())))
				continue
			}
			candles = append(candles, &newCandle)
		}
	}

	return candles, nil
}

// generateFetchKey calculates the correct badger key for the given market, interval and timestamp.
func (c *Candle) generateFetchKey(market string, interval types.Interval, since time.Time) []byte {
	// returns valid key for Market, interval and timestamp
	// round floor by integer division
	switch interval {
	case types.Interval_INTERVAL_I1M:
		fallthrough
	case types.Interval_INTERVAL_I5M:
		fallthrough
	case types.Interval_INTERVAL_I15M:
		fallthrough
	case types.Interval_INTERVAL_I1H:
		fallthrough
	case types.Interval_INTERVAL_I6H:
		fallthrough
	case types.Interval_INTERVAL_I1D:
		return c.badger.candleKey(market, interval, vegatime.RoundToNearest(since, interval).UnixNano())
	default:
		return nil
	}

}

// FetchLastCandle return the last candle store for a given market and interval
func (c *Candle) FetchLastCandle(marketID string, interval types.Interval) (*types.Candle, error) {
	var candle types.Candle
	key := c.badger.lastCandleKey(marketID, interval)
	err := c.badger.db.View(func(txn *badger.Txn) error {
		lastCandleItem, err := txn.Get(key)
		if err != nil {
			return err
		}
		candleKey, err := lastCandleItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		candleItem, err := txn.Get(candleKey)
		if err != nil {
			return err
		}
		candleBuf, err := candleItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		if err := proto.Unmarshal(candleBuf, &candle); err != nil {
			c.log.Error("Failed to unmarshal candle value from badger in candle store (FetchLastCandle)",
				logging.Error(err),
				logging.String("badger-key", string(key)))
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &candle, nil
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

			// Note: internal transport is per interval per market
			// SO we only notify for candle with specified interval and market
			if item.Candle.Interval != iT.Interval || item.Market != iT.Market {
				// Skip to next market/candle item
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
