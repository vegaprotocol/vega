package storage

import (
	"errors"
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
)

// Party is used for memory/RAM based parties storage.
type Party struct {
	*Config
	db map[string]types.Party
}

// NewStore returns a concrete implementation of a parties Store.
func NewParties(config *Config) (*Party, error) {
	return &Party{
		Config: config,
		db:     make(map[string]types.Party, 0),
	}, nil
}

// Post saves a given party to the mem-store.
func (ms *Party) Post(party *types.Party) error {
	if _, exists := ms.db[party.Name]; exists {
		return errors.New(fmt.Sprintf("party %s already exists in store", party.Name))
	}
	ms.db[party.Name] = *party
	return nil
}

// GetByID searches for the given party by id/name in the mem-store.
func (ms *Party) GetByID(id string) (*types.Party, error) {
	if _, exists := ms.db[id]; !exists {
		return nil, errors.New(fmt.Sprintf("party %s not found in store", id))
	}
	party := ms.db[id]
	return &party, nil
}

// GetAll returns all parties in the mem-store.
func (ms *Party) GetAll() ([]*types.Party, error) {
	res := make([]*types.Party, 0, len(ms.db))
	for k := range ms.db {
		kv := ms.db[k]
		res = append(res, &kv)
	}
	return res, nil
}
