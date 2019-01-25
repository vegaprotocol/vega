package markets

import (
	"vega/internal/storage"
	"vega/msg"
)

//Service provides the interface for markets business logic.
type Service interface {
	// AddMarket stores the given market.
	AddMarket(market *msg.Market) error
	// GetMarketByName searches for the given market by name.
	GetMarketByName(name string) (*msg.Market, error)
	// GetAllMarkets returns all markets.
	GetAllMarkets() ([]*msg.Market, error)
}

type service struct {
	*Config
	store storage.MarketStore
}

// NewService creates an market service with the necessary dependencies
func NewService(store storage.MarketStore) Service {
	config := NewConfig()
	return &service{
		config,
		store,
	}
}

// AddMarket stores the given market.
func (s *service) AddMarket(party *msg.Market) error {
	return s.store.Post(party)
}

// GetMarket searches for the given market by name.
func (s *service) GetMarketByName(name string) (*msg.Market, error) {
	p, err := s.store.GetByName(name)
	return p, err
}

// GetAllMarkets returns all markets.
func (s *service) GetAllMarkets() ([]*msg.Market, error) {
	p, err := s.store.GetAll()
	return p, err
}


