package storage

import (
	"vega/msg"
	"fmt"
	"github.com/pkg/errors"
)

// Store provides the data storage contract for markets.
type MarketStore interface {
	//Subscribe(markets chan<- []msg.Market) uint64
	//Unsubscribe(id uint64) error

	// Post adds a party to the store, this adds
	// to queue the operation to be committed later.
	Post(party *msg.Market) error

	// Commit typically saves any operations that are queued to underlying storage,
	// if supported by underlying storage implementation.
	Commit() error

	// Close can be called to clean up and close any storage
	// connections held by the underlying storage mechanism.
	Close()

	// GetByName searches for the given party by name in the underlying store.
	GetByName(name string) (*msg.Market, error)

	// GetAll returns all markets in the underlying store.
	GetAll() ([]*msg.Market, error)
}

// memMarketStore is used for memory/RAM based markets storage.
type memMarketStore struct {
	*Config
	db map[string]msg.Market
}

// NewStore returns a concrete implementation of a markets Store.
func NewMarketStore(config *Config) MarketStore {
	return &memMarketStore{
		Config: config,
		db:     make(map[string]msg.Market, 0),
	}
}

// Post saves a given market to the mem-store.
func (ms *memMarketStore) Post(market *msg.Market) error {
	if _, exists := ms.db[market.Name]; exists {
		return errors.New(fmt.Sprintf("market %s already exists in store", market.Name))
	}
	ms.db[market.Name] = *market
	return nil
}

// GetByName searches for the given market by name in the mem-store.
func (ms *memMarketStore) GetByName(name string) (*msg.Market, error) {
	if _, exists := ms.db[name]; !exists {
		return nil, errors.New(fmt.Sprintf("market %s not found in store", name))
	}
	market := ms.db[name]
	return &market, nil
}

// GetAll returns all markets in the mem-store.
// GetAll returns all markets in the mem-store.
func (ms *memMarketStore) GetAll() ([]*msg.Market, error) {
	res := make([]*msg.Market, len(ms.db))
	for _, v := range ms.db {
		res = append(res, &v)
	}
	return res, nil
}

// Commit typically saves any operations that are queued to underlying storage,
// if supported by underlying storage implementation.
func (ms *memMarketStore) Commit() error {
	// Not required with a mem-store implementation.
	return nil
}

// Close can be called to clean up and close any storage
// connections held by the underlying storage mechanism.
func (ms *memMarketStore) Close() {
	// Not required with a mem-store implementation.
}


