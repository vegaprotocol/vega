package storage

import (
	storcfg "code.vegaprotocol.io/vega/internal/storage/config"
	types "code.vegaprotocol.io/vega/proto"
)

// Risk is used for memory/RAM based risk storage.
type Risk struct {
	Config storcfg.RiskConfig
}

// NewRisks returns a concrete implementation of RiskStore.
func NewRisks(config storcfg.RiskConfig) (*Risk, error) {
	return &Risk{
		Config: config,
	}, nil
}

func (r *Risk) ReloadConf(config storcfg.RiskConfig) {
	// nothing to do for now
}

// Post saves a given risk factor to the mem-store.
func (ms *Risk) Post(risk *types.RiskFactor) error {
	return nil
}

// Put updates a given risk factor to the mem-store.
func (ms *Risk) Put(risk *types.RiskFactor) error {
	return nil
}

// GetByMarket searches for the risk factors relating to the market param in the mem-store.
func (ms *Risk) GetByMarket(name string) (*types.RiskFactor, error) {
	return &types.RiskFactor{
		Market: name,
		Long:   1,
		Short:  1,
	}, nil
}

// Commit typically saves any operations that are queued to underlying storage,
// if supported by underlying storage implementation.
func (ms *Risk) Commit() error {
	// No work required with a mem-store implementation.
	return nil
}

// Close can be called to clean up and close any storage
// connections held by the underlying storage mechanism.
func (ms *Risk) Close() error {
	// No work required with a mem-store implementation.
	return nil
}
