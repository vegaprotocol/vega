package storage

import (
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	ErrNoMarketDataForMarket = errors.New("no market data for market")
)

type MarketData struct {
	Config
	log *logging.Logger

	// market id to data
	store map[string]proto.MarketData
	mu    sync.RWMutex

	// subscriptions
	subscribers  map[uint64]chan<- []proto.MarketData
	subscriberID uint64
	subMu        sync.Mutex
}

// ReloadConf update the internal conf of the market
func (m *MarketData) ReloadConf(cfg Config) {
	m.log.Info("reloading configuration")
	if m.log.GetLevel() != cfg.Level.Get() {
		m.log.Info("updating log level",
			logging.String("old", m.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		m.log.SetLevel(cfg.Level.Get())
	}

	m.Config = cfg
}

func NewMarketData(log *logging.Logger, c Config) *MarketData {
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())
	return &MarketData{
		Config:      c,
		log:         log,
		store:       map[string]proto.MarketData{},
		subscribers: map[uint64]chan<- []proto.MarketData{},
	}
}

func (m *MarketData) GetByID(marketID string) (proto.MarketData, error) {
	m.mu.RLock()
	md, ok := m.store[marketID]
	m.mu.RUnlock()
	if !ok {
		return proto.MarketData{}, nil
	}
	return md, nil
}

func (m *MarketData) GetAll() []proto.MarketData {
	out := make([]proto.MarketData, 0, len(m.store))
	m.mu.RLock()
	for _, v := range m.store {
		out = append(out, v)
	}
	m.mu.RUnlock()
	return out
}

func (m *MarketData) SaveBatch(batch []proto.MarketData) {
	if len(batch) <= 0 {
		return
	}
	m.mu.Lock()
	for _, v := range batch {
		m.store[v.Market] = v
	}
	m.mu.Unlock()
	m.notify(batch)
}

func (m *MarketData) Subscribe(c chan<- []proto.MarketData) uint64 {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	m.subscriberID++
	m.subscribers[m.subscriberID] = c

	m.log.Debug("MarketData subscriber added in market data store",
		logging.Uint64("subscriber-id", m.subscriberID))

	return m.subscriberID
}

// Unsubscribe from account store updates.
func (m *MarketData) Unsubscribe(id uint64) error {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	if len(m.subscribers) == 0 {
		m.log.Debug("Un-subscribe called in market data store, no subscribers connected",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	if _, exists := m.subscribers[id]; exists {
		delete(m.subscribers, id)

		m.log.Debug("Un-subscribe called in market data store, subscriber removed",
			logging.Uint64("subscriber-id", id))

		return nil
	}

	m.log.Warn("Un-subscribe called in market data store, subscriber does not exist",
		logging.Uint64("subscriber-id", id))

	return fmt.Errorf("MarketData store subscriber does not exist with id: %d", id)
}

func (m *MarketData) notify(batch []proto.MarketData) {
	if len(batch) == 0 {
		return
	}

	m.subMu.Lock()
	if len(m.subscribers) == 0 {
		m.log.Debug("No subscribers connected in market data store")
		m.subMu.Unlock()
		return
	}

	var ok bool
	for id, sub := range m.subscribers {
		select {
		case sub <- batch:
			ok = true
		default:
			ok = false
		}
		if ok {
			m.log.Debug("MarketData channel updated for subscriber successfully",
				logging.Uint64("id", id))
		} else {
			m.log.Debug("MarketData channel could not be updated for subscriber",
				logging.Uint64("id", id))
		}
	}
	m.subMu.Unlock()
}
