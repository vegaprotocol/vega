package parties

import (
	"vega/internal/storage"
)

//Service provides the interface for parties business logic.
type Service interface {
	// CreateParty stores the given party.
	CreateParty(party *Party) error
	// GetByName searches for the given party by name.
	GetByName(name string) (*Party, error)
	// GetAll returns all parties.
	GetAll() ([]*Party, error)
}

type partyService struct {
	*Config
	store storage.PartyStore
}

// NewService creates a Parties service with the necessary dependencies
func NewService(store storage.PartyStore) Service {
	config := NewConfig()
	return &partyService{
		config,
		store,
	}
}

// CreateParty stores the given party.
func (s *partyService) CreateParty(party *Party) error {
	return s.store.Post(party)
}

// GetByName searches for the given party by name.
func (s *partyService) GetByName(name string) (*Party, error) {
	p, err := s.store.GetByName(name)
	return p, err
}

// GetAll returns all parties.
func (s *partyService) GetAll() ([]*Party, error) {
	p, err := s.store.GetAll()
	return p, err
}
