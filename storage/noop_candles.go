package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

// NoopCandle is a package internal data struct that implements the CandleStore interface.
type NoopCandle struct {
	Config

	cfgMu sync.Mutex
	log   *logging.Logger
	// badger       *badgerStore
	subscribers  map[uint64]*InternalTransport
	subscriberID uint64
	queue        []marketCandle
	mu           sync.Mutex
}

func NewNoopCandles(log *logging.Logger, c Config) *NoopCandle {
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	return &NoopCandle{
		log:    log,
		Config: c,
		// badger:      &bs,
		subscribers: make(map[uint64]*InternalTransport),
		queue:       make([]marketCandle, 0),
	}
}

// ReloadConf update the internal Candle configuration
func (c *NoopCandle) ReloadConf(cfg Config) {
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

func (c *NoopCandle) Subscribe(iT *InternalTransport) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.subscriberID++
	c.subscribers[c.subscriberID] = iT

	c.log.Debug("Candle subscriber added in candle store",
		logging.Uint64("subscriber-id", c.subscriberID))

	return c.subscriberID
}

func (c *NoopCandle) Unsubscribe(id uint64) error {
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

func (c *NoopCandle) Close() error {
	return nil
}

func (c *NoopCandle) GenerateCandlesFromBuffer(marketID string, buf map[string]types.Candle) error {
	return nil
}

func (c *NoopCandle) GetCandles(ctx context.Context, market string, since time.Time, interval types.Interval) ([]*types.Candle, error) {
	return []*types.Candle{}, nil
}

func (c *NoopCandle) FetchLastCandle(marketID string, interval types.Interval) (*types.Candle, error) {
	var candle types.Candle
	return &candle, nil
}

func (c *NoopCandle) notify() error {
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
