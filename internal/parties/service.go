package parties

import (
	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"
	"context"
)

//Service provides the interface for parties business logic.
type Service interface {
	// CreateParty stores the given party.
	CreateParty(ctx context.Context, party *types.Party) error
	// GetByName searches for the given party by name.
	GetByName(ctx context.Context, name string) (*types.Party, error)
	// GetAll returns all parties.
	GetAll(ctx context.Context) ([]*types.Party, error)
}

type partyService struct {
	*Config
	store storage.PartyStore
}

// NewPartyService creates a Parties service with the necessary dependencies
func NewPartyService(config *Config, store storage.PartyStore) (Service, error) {
	return &partyService{
		config,
		store,
	}, nil
}

// CreateParty stores the given party.
func (s *partyService) CreateParty(ctx context.Context, party *types.Party) error {
	return s.store.Post(party)
}

// GetByName searches for the given party by name.
func (s *partyService) GetByName(ctx context.Context, name string) (*types.Party, error) {
	p, err := s.store.GetByName(name)
	return p, err
}

// GetAll returns all parties.
func (s *partyService) GetAll(ctx context.Context) ([]*types.Party, error) {
	p, err := s.store.GetAll()
	return p, err
}
