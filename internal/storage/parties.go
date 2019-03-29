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
func NewPartyStore(config *Config) (*Party, error) {
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

// GetByName searches for the given party by name in the mem-store.
func (ms *Party) GetByName(name string) (*types.Party, error) {
	if _, exists := ms.db[name]; !exists {
		return nil, errors.New(fmt.Sprintf("party %s not found in store", name))
	}
	party := ms.db[name]
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

// Commit typically saves any operations that are queued to underlying storage,
// if supported by underlying storage implementation.
func (ms *Party) Commit() error {
	// Not required with a mem-store implementation.
	return nil
}

// Close can be called to clean up and close any storage
// connections held by the underlying storage mechanism.
func (ms *Party) Close() error {
	// Not required with a mem-store implementation.
	return nil
}
