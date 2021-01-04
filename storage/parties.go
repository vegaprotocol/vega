package storage

import (
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

// Party is used for memory/RAM based parties storage.
type Party struct {
	Config
	db map[string]types.Party
	mu sync.RWMutex
}

// NewParties returns a concrete implementation of a parties Store.
func NewParties(config Config) (*Party, error) {
	return &Party{
		Config: config,
		db:     make(map[string]types.Party),
	}, nil
}

// ReloadConf update the internal configuration of the party
func (p *Party) ReloadConf(config Config) {
	// nothing to do for now
}

// Post saves a given party to the mem-store.
func (p *Party) Post(party *types.Party) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.db[party.Id]; exists {
		return fmt.Errorf("party %s already exists in store", party.Id)
	}
	p.db[party.Id] = *party
	return nil
}

// GetByID searches for the given party by id/name in the mem-store.
func (p *Party) GetByID(id string) (party *types.Party, err error) {
	timer := metrics.NewTimeCounter("-", "partystore", "GetByID")

	p.mu.RLock()
	defer p.mu.RUnlock()

	pty, exists := p.db[id]
	if !exists {
		err = fmt.Errorf("party %s not found in store", id)
	} else {
		party = &pty
	}
	timer.EngineTimeCounterAdd()
	return
}

// GetAll returns all parties in the mem-store.
func (p *Party) GetAll() ([]*types.Party, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	res := make([]*types.Party, 0, len(p.db))
	for k := range p.db {
		kv := p.db[k]
		res = append(res, &kv)
	}
	return res, nil
}

type SaveBatchError struct {
	parties []string
}

func (s SaveBatchError) Error() string {
	return fmt.Sprintf("parties already exists: %v", s.parties)
}

func (p *Party) SaveBatch(batch []types.Party) error {
	var sberr SaveBatchError
	for _, v := range batch {
		err := p.Post(&v)
		if err != nil {
			sberr.parties = append(sberr.parties, v.Id)
		}
	}
	if len(sberr.parties) > 0 {
		return sberr
	}

	return nil
}
