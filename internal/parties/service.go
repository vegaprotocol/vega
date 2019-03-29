package parties

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/part_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/parties PartyStore
type PartyStore interface {
	Post(party *types.Party) error
	GetByName(name string) (*types.Party, error)
	GetAll() ([]*types.Party, error)
}

type Svc struct {
	*Config
	store PartyStore
}

// NewPartyService creates a Parties service with the necessary dependencies
func NewPartyService(config *Config, store PartyStore) (*Svc, error) {
	return &Svc{
		Config: config,
		store:  store,
	}, nil
}

// CreateParty stores the given party.
func (s *Svc) CreateParty(ctx context.Context, party *types.Party) error {
	return s.store.Post(party)
}

// GetByName searches for the given party by name.
func (s *Svc) GetByName(ctx context.Context, name string) (*types.Party, error) {
	return s.store.GetByName(name)
}

// GetAll returns all parties.
func (s *Svc) GetAll(ctx context.Context) ([]*types.Party, error) {
	return s.store.GetAll()
}
