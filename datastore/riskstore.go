package datastore

import "vega/msg"

type memRiskStore struct {
	store *MemStore
}

// NewRiskStore initialises a new RiskStore backed by a MemStore.
func NewRiskStore(ms *MemStore) RiskStore {
	return &memRiskStore{store: ms}
}

func (m *memRiskStore) GetMarginByParty(party string) ([]*msg.Margin, error) {
	return nil, nil
}
