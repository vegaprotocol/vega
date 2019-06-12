package storage

import (
	"errors"
	"fmt"
	"sync"

	storcfg "code.vegaprotocol.io/vega/internal/storage/config"
	types "code.vegaprotocol.io/vega/proto"
)

// Party is used for memory/RAM based parties storage.
type Party struct {
	Config storcfg.PartiesConfig
	db     map[string]types.Party
	mu     sync.RWMutex
}

// NewStore returns a concrete implementation of a parties Store.
func NewParties(config storcfg.PartiesConfig) (*Party, error) {
	return &Party{
		Config: config,
		db:     make(map[string]types.Party, 0),
	}, nil
}

func (p *Party) ReloadConf(config storcfg.PartiesConfig) {
	// nothing to do for now
}

// Post saves a given party to the mem-store.
func (ms *Party) Post(party *types.Party) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if _, exists := ms.db[party.Id]; exists {
		return errors.New(fmt.Sprintf("party %s already exists in store", party.Id))
	}
	ms.db[party.Id] = *party
	return nil
}

// GetByID searches for the given party by id/name in the mem-store.
func (ms *Party) GetByID(id string) (*types.Party, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if _, exists := ms.db[id]; !exists {
		return nil, errors.New(fmt.Sprintf("party %s not found in store", id))
	}
	party := ms.db[id]
	return &party, nil
}

// GetAll returns all parties in the mem-store.
func (ms *Party) GetAll() ([]*types.Party, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	res := make([]*types.Party, 0, len(ms.db))
	for k := range ms.db {
		kv := ms.db[k]
		res = append(res, &kv)
	}
	return res, nil
}
