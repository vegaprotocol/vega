package storage

import (
	"fmt"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

// Store provides the data storage contract for markets.
//go:generate go run github.com/golang/mock/mockgen -destination newmocks/market_store_mock.go -package newmocks code.vegaprotocol.io/vega/internal/storage MarketStore
type MarketStore interface {
	//Subscribe(markets chan<- []types.Market) uint64
	//Unsubscribe(id uint64) error

	// Post adds a market to the store, this adds
	// to queue the operation to be committed later.
	Post(party *types.Market) error

	// Commit typically saves any operations that are queued to underlying storage,
	// if supported by underlying storage implementation.
	Commit() error

	// Close can be called to clean up and close any storage
	// connections held by the underlying storage mechanism.
	Close() error

	// GetByID searches for the given market by id in the underlying store.
	GetByID(name string) (*types.Market, error)

	// GetAll returns all markets in the underlying store.
	GetAll() ([]*types.Market, error)
}

// memMarketStore is used for memory/RAM based markets storage.
type memMarketStore struct {
	*Config
	db map[string]types.Market
}

// NewMarketStore returns a concrete implementation of MarketStore.
func NewMarketStore(config *Config) (MarketStore, error) {
	return &memMarketStore{
		Config: config,
		db:     make(map[string]types.Market, 0),
	}, nil
}

// Post saves a given market to the mem-store.
func (ms *memMarketStore) Post(market *types.Market) error {
	if _, exists := ms.db[market.Id]; exists {
		return errors.New(fmt.Sprintf("market %s already exists in store", market.Id))
	}
	ms.db[market.Id] = *market
	return nil
}

// GetByID searches for the given market by id in the mem-store.
func (ms *memMarketStore) GetByID(id string) (*types.Market, error) {
	if _, exists := ms.db[id]; !exists {
		return nil, errors.New(fmt.Sprintf("market %s not found in store", id))
	}
	market := ms.db[id]
	return &market, nil
}

// GetAll returns all markets in the mem-store.
func (ms *memMarketStore) GetAll() ([]*types.Market, error) {
	res := make([]*types.Market, 0, len(ms.db))
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
func (ms *memMarketStore) Close() error {
	// Not required with a mem-store implementation.
	return nil
}
