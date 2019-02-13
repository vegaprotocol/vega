package parties

import (
	"vega/internal/storage"
	types "vega/proto"
)

//Service provides the interface for parties business logic.
type Service interface {
	// CreateParty stores the given party.
	CreateParty(party *types.Party) error
	// GetByName searches for the given party by name.
	GetByName(name string) (*types.Party, error)
	// GetAll returns all parties.
	GetAll() ([]*types.Party, error)
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
func (s *partyService) CreateParty(party *types.Party) error {
	return s.store.Post(party)
}

// GetByName searches for the given party by name.
func (s *partyService) GetByName(name string) (*types.Party, error) {
	p, err := s.store.GetByName(name)
	return p, err
}

// GetAll returns all parties.
func (s *partyService) GetAll() ([]*types.Party, error) {
	p, err := s.store.GetAll()
	return p, err
}
