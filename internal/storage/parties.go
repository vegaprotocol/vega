package storage

import (
	"errors"
	"fmt"
	"vega/msg"
)

// Store provides the data storage contract for parties.
type PartyStore interface {
	//Subscribe(parties chan<- []parties.Party) uint64
	//Unsubscribe(id uint64) error

	// Post adds a party to the store, this adds
	// to queue the operation to be committed later.
	Post(party *msg.Party) error

	// Commit typically saves any operations that are queued to underlying storage,
	// if supported by underlying storage implementation.
	Commit() error

	// Close can be called to clean up and close any storage
	// connections held by the underlying storage mechanism.
	Close() error

	// GetByName searches for the given party by name in the underlying store.
	GetByName(name string) (*msg.Party, error)

	// GetAll returns all parties in the underlying store.
	GetAll() ([]*msg.Party, error)
}

// memPartyStore is used for memory/RAM based parties storage.
type memPartyStore struct {
	*Config
	db map[string]msg.Party
}

// NewStore returns a concrete implementation of a parties Store.
func NewPartyStore(config *Config) (PartyStore, error) {
	return &memPartyStore{
		Config: config,
		db:     make(map[string]msg.Party, 0),
	}, nil
}

// Post saves a given party to the mem-store.
func (ms *memPartyStore) Post(party *msg.Party) error {
	if _, exists := ms.db[party.Name]; exists {
		return errors.New(fmt.Sprintf("party %s already exists in store", party.Name))
	}
	ms.db[party.Name] = *party
	return nil
}

// GetByName searches for the given party by name in the mem-store.
func (ms *memPartyStore) GetByName(name string) (*msg.Party, error) {
	if _, exists := ms.db[name]; !exists {
		return nil, errors.New(fmt.Sprintf("party %s not found in store", name))
	}
	party := ms.db[name]
	return &party, nil
}

// GetAll returns all parties in the mem-store.
func (ms *memPartyStore) GetAll() ([]*msg.Party, error) {
	res := make([]*msg.Party, len(ms.db))
	for _, v := range ms.db {
		res = append(res, &v)
	}
	return res, nil
}

// Commit typically saves any operations that are queued to underlying storage,
// if supported by underlying storage implementation.
func (ms *memPartyStore) Commit() error {
	// Not required with a mem-store implementation.
	return nil
}

// Close can be called to clean up and close any storage
// connections held by the underlying storage mechanism.
func (ms *memPartyStore) Close() error {
	// Not required with a mem-store implementation.
	return nil
}
