package storage

import (
	types "vega/proto"
)

// Store provides the data storage contract for risk factors.
type RiskStore interface {
	//Subscribe(riskFactors chan<- []types.RiskFactor) uint64
	//Unsubscribe(id uint64) error

	// Post adds a risk factor to the store, this adds
	// to queue the operation to be committed later.
	Post(risk *types.RiskFactor) error

	// Put updates a risk factor in the store, adds
	// to queue the operation to be committed later.
	Put(risk *types.RiskFactor) error

	// Commit typically saves any operations that are queued to underlying storage,
	// if supported by underlying storage implementation.
	Commit() error

	// Close can be called to clean up and close any storage
	// connections held by the underlying storage mechanism.
	Close() error

	// GetByMarket searches for the given risk factor by market in the underlying store.
	GetByMarket(market string) (*types.RiskFactor, error)
}

// memMarketStore is used for memory/RAM based markets storage.
type memRiskStore struct {
	*Config
}

// NewRiskStore returns a concrete implementation of RiskStore.
func NewRiskStore(config *Config) (RiskStore, error) {
	return &memRiskStore{
		Config: config,
	}, nil
}

// Post saves a given risk factor to the mem-store.
func (ms *memRiskStore) Post(risk *types.RiskFactor) error {
	return nil
}

// Put updates a given risk factor to the mem-store.
func (ms *memRiskStore) Put(risk *types.RiskFactor) error {
	return nil
}

// GetByMarket searches for the risk factors relating to the market param in the mem-store.
func (ms *memRiskStore) GetByMarket(name string) (*types.RiskFactor, error) {
	return &types.RiskFactor{
		Market: name,
		Long:   1,
		Short:  1,
	}, nil
}

// Commit typically saves any operations that are queued to underlying storage,
// if supported by underlying storage implementation.
func (ms *memRiskStore) Commit() error {
	// No work required with a mem-store implementation.
	return nil
}

// Close can be called to clean up and close any storage
// connections held by the underlying storage mechanism.
func (ms *memRiskStore) Close() error {
	// No work required with a mem-store implementation.
	return nil
}
