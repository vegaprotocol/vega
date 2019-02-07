package markets

import (
	"vega/internal/storage"
	types "vega/proto"
	"context"
	"vega/internal/logging"
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
	GetDepth(ctx context.Context, market string) (marketDepth types.MarketDepth, err error)
	// ObserveMarket provides a way to listen to changes on VEGA markets.
	ObserveMarkets(ctx context.Context) (markets <-chan []types.Market, ref uint64)
	// ObserveDepth provides a way to listen to changes on the Depth of Market for a given market.
	ObserveDepth(ctx context.Context, market string) (depth <-chan types.MarketDepth, ref uint64)
}

type marketService struct {
	*Config
	marketStore storage.MarketStore
	orderStore storage.OrderStore
}

// NewService creates an market service with the necessary dependencies
func NewService(config *Config, marketStore storage.MarketStore, orderStore storage.OrderStore) Service {
	return &marketService{
		config,
		marketStore,
		orderStore,
	}
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
func (s *marketService) GetDepth(ctx context.Context, market string) (marketDepth types.MarketDepth, err error) {
	m, err := s.marketStore.GetByName(market)
	if err != nil {
		return types.MarketDepth{}, err
	}
	return s.orderStore.GetMarketDepth(m.Name)
}

// ObserveDepth provides a way to listen to changes on the Depth of Market for a given market.
func (s *marketService) ObserveDepth(ctx context.Context, market string) (<-chan types.MarketDepth, uint64) {
	depth := make(chan types.MarketDepth)
	internal := make(chan []types.Order)
	ref := s.orderStore.Subscribe(internal)

	go func(id uint64, internal chan []types.Order, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		<-ctx.Done()
		s.log.Debugf("MarketService -> depth closed connection: %d [%s]", id, ip)
		err := s.orderStore.Unsubscribe(id)
		if err != nil {
			s.log.Errorf("Error un-subscribing depth when context.Done() on MarketService for subscriber %d [%s]: %s", id, ip, err)
		}
	}(ref, internal, ctx)

	go func(id uint64, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		for range internal {
			d, err := s.orderStore.GetMarketDepth(market)
			if err != nil {
				s.log.Errorf("Error calculating market depth for subscriber %d [%s]: %s", ref, ip, err)
			} else {
				select {
				case depth <- d:
					s.log.Debugf("MarketService -> depth for subscriber %d [%s] sent successfully", ref, ip)
				default:
					s.log.Debugf("MarketService -> depth for subscriber %d [%s] not sent", ref, ip)
				}
			}
		}
		s.log.Debugf("MarketService -> Channel for depth subscriber %d [%s] has been closed", ref, ip)
	}(ref, ctx)

	return depth, ref
}

func (s *marketService) ObserveMarkets(ctx context.Context) (markets <-chan []types.Market, ref uint64) {
	 return nil, 0
}

