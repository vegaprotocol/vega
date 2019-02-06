package parties

import (
	"vega/internal/storage"
	"vega/msg"
)

//Service provides the interface for parties business logic.
type Service interface {
	// CreateParty stores the given party.
	CreateParty(party *msg.Party) error
	// GetByName searches for the given party by name.
	GetByName(name string) (*msg.Party, error)
	// GetAll returns all parties.
	GetAll() ([]*msg.Party, error)
}

type partyService struct {
	*Config
	store storage.PartyStore
}

// NewPartyService creates a Parties service with the necessary dependencies
func NewPartyService(store storage.PartyStore) Service {
	config := NewConfig()
	return &partyService{
		config,
		store,
	}
}

// CreateParty stores the given party.
func (s *partyService) CreateParty(party *msg.Party) error {
	return s.store.Post(party)
}

// GetByName searches for the given party by name.
func (s *partyService) GetByName(name string) (*msg.Party, error) {
	p, err := s.store.GetByName(name)
	return p, err
}

// GetAll returns all parties.
func (s *partyService) GetAll() ([]*msg.Party, error) {
	p, err := s.store.GetAll()
	return p, err
}
