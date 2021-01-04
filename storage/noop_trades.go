package storage

import (
	"context"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto/gen/golang"

	"github.com/pkg/errors"
)

type NoopTrade struct {
	Config

	cfgMu        sync.Mutex
	log          *logging.Logger
	subscribers  map[uint64]chan<- []types.Trade
	subscriberID uint64
	mu           sync.Mutex
}

func NewNoopTrades(log *logging.Logger, c Config) *NoopTrade {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	return &NoopTrade{
		log:         log,
		Config:      c,
		subscribers: make(map[uint64]chan<- []types.Trade),
	}
}

func (ts *NoopTrade) ReloadConf(cfg Config) {
	ts.log.Info("reloading configuration")
	if ts.log.GetLevel() != cfg.Level.Get() {
		ts.log.Info("updating log level",
			logging.String("old", ts.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		ts.log.SetLevel(cfg.Level.Get())
	}

	// only Timeout is really use in here
	ts.cfgMu.Lock()
	ts.Config = cfg
	ts.cfgMu.Unlock()
}

func (ts *NoopTrade) Subscribe(trades chan<- []types.Trade) uint64 {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.subscriberID++
	ts.subscribers[ts.subscriberID] = trades

	ts.log.Debug("Trades subscriber added in order store",
		logging.Uint64("subscriber-id", ts.subscriberID))

	return ts.subscriberID
}

func (ts *NoopTrade) Unsubscribe(id uint64) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if len(ts.subscribers) == 0 {
		ts.log.Debug("Un-subscribe called in trade store, no subscribers connected",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	if _, exists := ts.subscribers[id]; exists {
		delete(ts.subscribers, id)
		ts.log.Debug("Un-subscribe called in trade store, subscriber removed",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	return fmt.Errorf("subscriber to Trades store does not exist with id: %d", id)
}

func (ts *NoopTrade) Post(trade *types.Trade) error {
	return nil
}

func (ts *NoopTrade) Commit() (err error) {
	return
}

func (ts *NoopTrade) GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) ([]*types.Trade, error) {
	return []*types.Trade{}, nil
}

func (ts *NoopTrade) GetByMarketAndID(ctx context.Context, market string, id string) (*types.Trade, error) {
	var trade types.Trade
	return &trade, nil
}

func (ts *NoopTrade) GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, market *string) ([]*types.Trade, error) {
	return []*types.Trade{}, nil
}

func (ts *NoopTrade) GetByPartyAndID(ctx context.Context, party string, id string) (*types.Trade, error) {
	var trade types.Trade
	return &trade, nil
}

func (ts *NoopTrade) GetByOrderID(ctx context.Context, orderID string, skip, limit uint64, descending bool, market *string) ([]*types.Trade, error) {
	return []*types.Trade{}, nil
}

func (ts *NoopTrade) GetMarkPrice(ctx context.Context, market string) (uint64, error) {
	recentTrade, err := ts.GetByMarket(ctx, market, 0, 1, true)
	if err != nil {
		return 0, err
	}

	if len(recentTrade) == 0 {
		return 0, errors.New("no trades available when getting market price")
	}

	return recentTrade[0].Price, nil
}

func (ts *NoopTrade) Close() error {
	return nil
}

func (ts *NoopTrade) GetTradesBySideBuckets(ctx context.Context, party string) map[string]*MarketBucket {
	return map[string]*MarketBucket{}
}

func (ts *NoopTrade) SaveBatch(batch []types.Trade) error {
	return nil
}
