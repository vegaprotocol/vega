package markets

import (
	"context"
	"sync/atomic"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/markets MarketStore
type MarketStore interface {
	Post(party *types.Market) error
	GetByID(name string) (*types.Market, error)
	GetAll() ([]*types.Market, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/markets OrderStore
type OrderStore interface {
	Subscribe(orders chan<- []types.Order) uint64
	Unsubscribe(id uint64) error
	GetMarketDepth(ctx context.Context, market string) (*types.MarketDepth, error)
}

type Svc struct {
	Config
	log            *logging.Logger
	marketStore    MarketStore
	orderStore     OrderStore
	subscribersCnt int32
}

// NewService creates an market service with the necessary dependencies
func NewService(log *logging.Logger, config Config, marketStore MarketStore, orderStore OrderStore) (*Svc, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Svc{
		log:         log,
		Config:      config,
		marketStore: marketStore,
		orderStore:  orderStore,
	}, nil
}

func (s *Svc) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.Config = cfg
}

// CreateMarket stores the given market.
func (s *Svc) CreateMarket(ctx context.Context, party *types.Market) error {
	return s.marketStore.Post(party)
}

// GetByID searches for the given market by name.
func (s *Svc) GetByID(ctx context.Context, id string) (*types.Market, error) {
	p, err := s.marketStore.GetByID(id)
	return p, err
}

// GetAll returns all markets.
func (s *Svc) GetAll(ctx context.Context) ([]*types.Market, error) {
	p, err := s.marketStore.GetAll()
	return p, err
}

// GetDepth returns the market depth for the given market.
func (s *Svc) GetDepth(ctx context.Context, marketID string) (marketDepth *types.MarketDepth, err error) {
	m, err := s.marketStore.GetByID(marketID)
	if err != nil {
		return nil, err
	}

	return s.orderStore.GetMarketDepth(ctx, m.Id)
}

func (s *Svc) GetMarketDepthSubscribersCount() int32 {
	return atomic.LoadInt32(&s.subscribersCnt)
}

// ObserveDepth provides a way to listen to changes on the Depth of Market for a given market.
func (s *Svc) ObserveDepth(ctx context.Context, retries int, market string) (<-chan *types.MarketDepth, uint64) {
	depth := make(chan *types.MarketDepth)
	internal := make(chan []types.Order)
	ref := s.orderStore.Subscribe(internal)

	retryCount := retries
	go func() {
		atomic.AddInt32(&s.subscribersCnt, 1)
		defer atomic.AddInt32(&s.subscribersCnt, -1)
		ip := logging.IPAddressFromContext(ctx)
		ctx, cfunc := context.WithCancel(ctx)
		defer cfunc()
		for {
			select {
			case <-ctx.Done():
				s.log.Debug(
					"Market depth subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				if err := s.orderStore.Unsubscribe(ref); err != nil {
					s.log.Error(
						"Failure un-subscribing market depth subscriber when context.Done()",
						logging.Uint64("id", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
				}
				close(internal)
				close(depth)
				return
			case <-internal: // we don't need the orders, we just need to know there was a change
				d, err := s.orderStore.GetMarketDepth(ctx, market)
				if err != nil {
					s.log.Debug(
						"Failure calculating market depth for subscriber",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
					continue
				}
				select {
				case depth <- d:
					retryCount = retries
					s.log.Debug(
						"Market depth for subscriber sent successfully",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
					)
				default:
					retryCount--
					if retryCount == 0 {
						s.log.Warn(
							"Market depth subscriber has hit the retry limit",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
							logging.Int("retries", retries),
						)
						cfunc()
					}
				}
			}
		}
	}()

	return depth, ref
}

func (s *Svc) ObserveMarkets(ctx context.Context) (markets <-chan []types.Market, ref uint64) {
	return nil, 0
}
