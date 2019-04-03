package parties

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/part_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/parties PartyStore
type PartyStore interface {
	Post(party *types.Party) error
	GetByID(id string) (*types.Party, error)
	GetAll() ([]*types.Party, error)
}

type Svc struct {
	*Config
	store PartyStore
}

// NewService creates a Parties service with the necessary dependencies
func NewService(config *Config, store PartyStore) (*Svc, error) {
	return &Svc{
		Config: config,
		store:  store,
	}, nil
}

// CreateParty stores the given party.
func (s *Svc) CreateParty(ctx context.Context, party *types.Party) error {
	return s.store.Post(party)
}

// GetByID searches for the given party by id.
func (s *Svc) GetByID(ctx context.Context, name string) (*types.Party, error) {
	return s.store.GetByID(name)
}

// GetAll returns all parties.
func (s *Svc) GetAll(ctx context.Context) ([]*types.Party, error) {
	return s.store.GetAll()
}
