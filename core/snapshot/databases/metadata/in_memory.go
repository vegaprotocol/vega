package metadata

type InMemoryAdapter struct {
	store map[string][]byte
}

func (a *InMemoryAdapter) Save(version []byte, state []byte) error {
	a.store[string(version)] = state
	return nil
}

func (a *InMemoryAdapter) Load(version []byte) (state []byte, err error) {
	s, ok := a.store[string(version)]
	if !ok {
		return nil, noMetadataForSnapshotVersion(version)
	}
	return s, nil
}

func (a *InMemoryAdapter) Close() error {
	return nil
}

func (a *InMemoryAdapter) Clear() error {
	a.store = map[string][]byte{}
	return nil
}

func (a *InMemoryAdapter) ContainsMetadata() bool {
	return len(a.store) > 0
}

func NewInMemoryAdapter() *InMemoryAdapter {
	return &InMemoryAdapter{
		store: map[string][]byte{},
	}
}
