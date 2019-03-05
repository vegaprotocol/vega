package markets

import (
	"context"

	"vega/internal/logging"
	"vega/internal/storage"
	types "vega/proto"
)

//Service provides the interface for VEGA markets business logic.
type Service interface {
	// CreateMarket stores the given market.
	CreateMarket(ctx context.Context, market *types.Market) error
	// GetByName searches for the given market by name.
	GetByName(ctx context.Context, name string) (*types.Market, error)
	// GetAll returns all markets.
	GetAll(ctx context.Context) ([]*types.Market, error)
	// GetDepth returns the market depth for the given market.
	GetDepth(ctx context.Context, market string) (marketDepth *types.MarketDepth, err error)
	// ObserveMarket provides a way to listen to changes on VEGA markets.
	ObserveMarkets(ctx context.Context) (markets <-chan []types.Market, ref uint64)
	// ObserveDepth provides a way to listen to changes on the Depth of Market for a given market.
	ObserveDepth(ctx context.Context, market string) (depth <-chan *types.MarketDepth, ref uint64)
}

type marketService struct {
	*Config
	marketStore storage.MarketStore
	orderStore  storage.OrderStore
}

// NewMarketService creates an market service with the necessary dependencies
func NewMarketService(config *Config, marketStore storage.MarketStore, orderStore storage.OrderStore) (Service, error) {
	return &marketService{
		config,
		marketStore,
		orderStore,
	}, nil
}

// CreateMarket stores the given market.
func (s *marketService) CreateMarket(ctx context.Context, party *types.Market) error {
	return s.marketStore.Post(party)
}

// GetByName searches for the given market by name.
func (s *marketService) GetByName(ctx context.Context, name string) (*types.Market, error) {
	p, err := s.marketStore.GetByName(name)
	return p, err
}

// GetAll returns all markets.
func (s *marketService) GetAll(ctx context.Context) ([]*types.Market, error) {
	p, err := s.marketStore.GetAll()
	return p, err
}

// GetDepth returns the market depth for the given market.
func (s *marketService) GetDepth(ctx context.Context, market string) (marketDepth *types.MarketDepth, err error) {
	m, err := s.marketStore.GetByName(market)
	if err != nil {
		return nil, err
	}
	return s.orderStore.GetMarketDepth(m.Name)
}

// ObserveDepth provides a way to listen to changes on the Depth of Market for a given market.
func (s *marketService) ObserveDepth(ctx context.Context, market string) (<-chan *types.MarketDepth, uint64) {
	depth := make(chan *types.MarketDepth)
	internal := make(chan []types.Order)
	ref := s.orderStore.Subscribe(internal)

	go func(id uint64, internal chan []types.Order, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		<-ctx.Done()
		s.log.Debug("Market depth subscriber closed connection",
			logging.Uint64("id", id),
			logging.String("ip-address", ip))
		err := s.orderStore.Unsubscribe(id)
		if err != nil {
			s.log.Error("Failure un-subscribing market depth subscriber when context.Done()",
				logging.Uint64("id", id),
				logging.String("ip-address", ip),
				logging.Error(err))
		}
	}(ref, internal, ctx)

	go func(id uint64, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		for range internal {
			d, err := s.orderStore.GetMarketDepth(market)
			if err != nil {
				s.log.Error("Failure calculating market depth for subscriber",
					logging.Uint64("ref", ref),
					logging.String("ip-address", ip),
					logging.Error(err))
			} else {
				select {
				case depth <- d:
					s.log.Debug("Market depth for subscriber sent successfully",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip))
				default:
					s.log.Debug("Market depth for subscriber not sent",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip))
				}
			}
		}
		s.log.Debug("Market depth subscriber channel has been closed",
			logging.Uint64("ref", ref),
			logging.String("ip-address", ip))
	}(ref, ctx)

	return depth, ref
}

func (s *marketService) ObserveMarkets(ctx context.Context) (markets <-chan []types.Market, ref uint64) {
	return nil, 0
}
