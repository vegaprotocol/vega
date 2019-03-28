package parties

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
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

//go:generate go run github.com/golang/mock/mockgen -destination newmocks/part_store_mock.go -package newmocks code.vegaprotocol.io/vega/internal/parties PartyStore
type PartyStore interface {
	Post(party *types.Party) error
	GetByName(name string) (*types.Party, error)
	GetAll() ([]*types.Party, error)
}

type partyService struct {
	*Config
	store PartyStore
}

// NewPartyService creates a Parties service with the necessary dependencies
func NewPartyService(config *Config, store PartyStore) (Service, error) {
	return &partyService{
		Config: config,
		store:  store,
	}, nil
}

// CreateParty stores the given party.
func (s *partyService) CreateParty(ctx context.Context, party *types.Party) error {
	return s.store.Post(party)
}

// GetByName searches for the given party by name.
func (s *partyService) GetByName(ctx context.Context, name string) (*types.Party, error) {
	return s.store.GetByName(name)
}

// GetAll returns all parties.
func (s *partyService) GetAll(ctx context.Context) ([]*types.Party, error) {
	return s.store.GetAll()
}
