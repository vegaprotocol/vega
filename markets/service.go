package markets

import (
	"context"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

// MarketStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_store_mock.go -package mocks code.vegaprotocol.io/vega/markets MarketStore
type MarketStore interface {
	Post(party *types.Market) error
	GetByID(name string) (*types.Market, error)
	GetAll() ([]*types.Market, error)
}

// MarketDataStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_data_store_mock.go -package mocks code.vegaprotocol.io/vega/markets MarketDataStore
type MarketDataStore interface {
	GetByID(string) (types.MarketData, error)
	GetAll() []types.MarketData
	Subscribe(chan<- []types.MarketData) uint64
	Unsubscribe(uint64) error
}

// OrderStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_store_mock.go -package mocks code.vegaprotocol.io/vega/markets OrderStore
type OrderStore interface {
	Subscribe(orders chan<- []types.Order) uint64
	Unsubscribe(id uint64) error
}

// MarketDepth ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_depth_mock.go -package mocks code.vegaprotocol.io/vega/markets MarketDepth
type MarketDepth interface {
	GetMarketDepth(ctx context.Context, market string, limit uint64) (*types.MarketDepth, error)
	Subscribe(orders chan<- *types.MarketDepthUpdate) uint64
	Unsubscribe(id uint64) error
}

// Svc represent the market service
type Svc struct {
	Config
	log             *logging.Logger
	marketStore     MarketStore
	orderStore      OrderStore
	marketDataStore MarketDataStore
	marketDepth     MarketDepth

	subscribersCnt           int32
	marketDataSubscribersCnt int32
}

// NewService creates an market service with the necessary dependencies
func NewService(
	log *logging.Logger,
	config Config,
	marketStore MarketStore,
	orderStore OrderStore,
	marketDataStore MarketDataStore,
	marketDepth MarketDepth,
) (*Svc, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Svc{
		log:             log,
		Config:          config,
		marketStore:     marketStore,
		orderStore:      orderStore,
		marketDataStore: marketDataStore,
		marketDepth:     marketDepth,
	}, nil
}

// ReloadConf update the market service internal configuration
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
func (s *Svc) GetDepth(ctx context.Context, marketID string, limit uint64) (marketDepth *types.MarketDepth, err error) {
	m, err := s.marketStore.GetByID(marketID)
	if err != nil {
		return nil, err
	}

	return s.marketDepth.GetMarketDepth(ctx, m.Id, limit)
}

// GetMarketDataByID return the market data for a given market
func (s *Svc) GetMarketDataByID(marketID string) (types.MarketData, error) {
	return s.marketDataStore.GetByID(marketID)
}

// GetMarketsData return the market data for a given market
func (s *Svc) GetMarketsData() []types.MarketData {
	return s.marketDataStore.GetAll()
}

// GetMarketDepthSubscribersCount return the number of subscribers to the
// market depths updates
func (s *Svc) GetMarketDepthSubscribersCount() int32 {
	return atomic.LoadInt32(&s.subscribersCnt)
}

// ObserveDepth provides a way to listen to changes on the Depth of Market for a given market.
func (s *Svc) ObserveDepth(ctx context.Context, retries int, market string) (<-chan *types.MarketDepth, uint64) {
	depth := make(chan *types.MarketDepth)
	internal := make(chan []types.Order)
	ref := s.orderStore.Subscribe(internal)

	var cancel func()
	ctx, cancel = context.WithCancel(ctx)
	go func() {
		atomic.AddInt32(&s.subscribersCnt, 1)
		defer atomic.AddInt32(&s.subscribersCnt, -1)
		ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
		defer cancel()
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
				d, err := s.marketDepth.GetMarketDepth(ctx, market, 0)
				if err != nil {
					s.log.Debug(
						"Failure calculating market depth for subscriber",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
					continue
				}
				retryCount := retries
				success := false
				for !success && retryCount >= 0 {
					select {
					case depth <- d:
						s.log.Debug(
							"Market depth for subscriber sent successfully",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
						)
						success = true
					default:
						retryCount--
						if retryCount >= 0 {
							s.log.Debug(
								"Market depth for subscriber not sent",
								logging.Uint64("ref", ref),
								logging.String("ip-address", ip),
							)
							time.Sleep(time.Duration(10) * time.Millisecond)
						}
					}
				}
				if !success && retryCount <= 0 {
					s.log.Warn(
						"Market depth subscriber has hit the retry limit",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
						logging.Int("retries", retries),
					)
					cancel()
				}
			}
		}
	}()

	return depth, ref
}

// ObserveDepthUpdates provides a way to listen to changes on the Depth of Market for a given market.
func (s *Svc) ObserveDepthUpdates(ctx context.Context, retries int, market string) (<-chan *types.MarketDepthUpdate, uint64) {
	depth := make(chan *types.MarketDepthUpdate)
	internal := make(chan *types.MarketDepthUpdate)
	ref := s.marketDepth.Subscribe(internal)

	var cancel func()
	ctx, cancel = context.WithCancel(ctx)
	go func() {
		atomic.AddInt32(&s.subscribersCnt, 1)
		defer atomic.AddInt32(&s.subscribersCnt, -1)
		ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				s.log.Debug(
					"Market depth updates subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				if err := s.marketDepth.Unsubscribe(ref); err != nil {
					if s.log.GetLevel() == logging.DebugLevel {
						s.log.Debug(
							"Failure un-subscribing market depth updates subscriber when context.Done()",
							logging.Uint64("id", ref),
							logging.String("ip-address", ip),
							logging.Error(err))
					}
				}
				close(internal)
				close(depth)
				return
			case update := <-internal:
				retryCount := retries
				success := false
				for !success && retryCount >= 0 {
					select {
					case depth <- update:
						s.log.Debug(
							"Market depth update for subscriber sent successfully",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
						)
						success = true
					default:
						retryCount--
						if retryCount >= 0 {
							s.log.Debug(
								"Market depth update for subscriber not sent",
								logging.Uint64("ref", ref),
								logging.String("ip-address", ip),
							)
							time.Sleep(time.Duration(10) * time.Millisecond)
						}
					}
				}
				if !success && retryCount <= 0 {
					if s.log.GetLevel() == logging.DebugLevel {
						s.log.Debug(
							"Market depth updates subscriber has hit the retry limit",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
							logging.Int("retries", retries))
					}
					cancel()
				}
			}
		}
	}()
	return depth, ref
}

func (s *Svc) ObserveMarketsData(
	ctx context.Context, retries int, marketID string,
) (<-chan []types.MarketData, uint64) {

	marketsDataCh := make(chan []types.MarketData)
	internal := make(chan []types.MarketData)
	ref := s.marketDataStore.Subscribe(internal)

	var cancel func()
	ctx, cancel = context.WithCancel(ctx)
	go func() {
		atomic.AddInt32(&s.marketDataSubscribersCnt, 1)
		defer atomic.AddInt32(&s.marketDataSubscribersCnt, -1)
		ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				s.log.Debug(
					"MarketData subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				// this error only happens when the subscriber reference doesn't exist
				// so we can still safely close the channels
				if err := s.marketDataStore.Unsubscribe(ref); err != nil {
					s.log.Error(
						"Failure un-subscribing market data subscriber when context.Done()",
						logging.Uint64("id", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
				}
				close(internal)
				close(marketsDataCh)
				return
			case mds := <-internal:
				filtered := make([]types.MarketData, 0, len(mds))
				for _, md := range mds {
					if len(marketID) <= 0 || marketID == md.Market {
						filtered = append(filtered, md)
					}
				}
				retryCount := retries
				success := false
				for !success && retryCount >= 0 {
					select {
					case marketsDataCh <- filtered:
						retryCount = retries
						s.log.Debug(
							"MarketsData for subscriber sent successfully",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
						)
						success = true
					default:
						retryCount--
						if retryCount > 0 {
							s.log.Debug(
								"MarketsData for subscriber not sent",
								logging.Uint64("ref", ref),
								logging.String("ip-address", ip))
						}
						time.Sleep(time.Duration(10) * time.Millisecond)
					}
				}
				if !success && retryCount <= 0 {
					s.log.Warn(
						"MarketsData subscriber has hit the retry limit",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
						logging.Int("retries", retries))
					cancel()
				}

			}
		}
	}()
	return marketsDataCh, ref
}

// GetMarketDataSubscribersCount return the number of subscribers to the
// market depths updates
func (s *Svc) GetMarketDataSubscribersCount() int32 {
	return atomic.LoadInt32(&s.marketDataSubscribersCnt)
}

// ObserveMarkets ...
func (s *Svc) ObserveMarkets(ctx context.Context) (markets <-chan []types.Market, ref uint64) {
	return nil, 0
}
