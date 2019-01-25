package parties

import (
	"vega/internal/storage"
)

//Service provides the interface for parties business logic.
type Service interface {
	// AddParty stores the given party.
	AddParty(party *Party) error
	// GetPartyByName searches for the given party by name.
	GetPartyByName(name string) (*Party, error)
	// GetAllParties returns all parties.
	GetAllParties() ([]*Party, error)
}

type service struct {
	*Config
	store storage.PartyStore
}

// NewService creates an adding service with the necessary dependencies
func NewService(store storage.PartyStore) Service {
	config := NewConfig()
	return &service{
		config,
		store,
	}
}

// AddParty stores the given party.
func (s *service) AddParty(party *Party) error {
	return s.store.Post(party)
}

// GetPartyByName searches for the given party by name.
func (s *service) GetPartyByName(name string) (*Party, error) {
	p, err := s.store.GetByName(name)
	return p, err
}

// GetAllParties returns all parties.
func (s *service) GetAllParties() ([]*Party, error) {
	p, err := s.store.GetAll()
	return p, err
}
