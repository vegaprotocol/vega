package storage

import (
	types "code.vegaprotocol.io/vega/proto"
)

// Risk is used for memory/RAM based risk storage.
type Risk struct {
	Config
}

// NewRisks returns a concrete implementation of RiskStore.
func NewRisks(config Config) (*Risk, error) {
	return &Risk{
		Config: config,
	}, nil
}

// ReloadConf update the internal configuration of the risk
func (r *Risk) ReloadConf(config Config) {
	// nothing to do for now
}

// Post saves a given risk factor to the mem-store.
func (r *Risk) Post(risk *types.RiskFactor) error {
	return nil
}

// Put updates a given risk factor to the mem-store.
func (r *Risk) Put(risk *types.RiskFactor) error {
	return nil
}

// GetByMarket searches for the risk factors relating to the market param in the mem-store.
func (r *Risk) GetByMarket(name string) (*types.RiskFactor, error) {
	return &types.RiskFactor{
		Market: name,
		Long:   1,
		Short:  1,
	}, nil
}

// Commit typically saves any operations that are queued to underlying storage,
// if supported by underlying storage implementation.
func (r *Risk) Commit() error {
	// No work required with a mem-store implementation.
	return nil
}

// Close can be called to clean up and close any storage
// connections held by the underlying storage mechanism.
func (r *Risk) Close() error {
	// No work required with a mem-store implementation.
	return nil
}
