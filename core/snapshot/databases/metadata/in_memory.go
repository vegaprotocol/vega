package metadata

import "errors"

var ErrUnknownSnapshotVersion = errors.New("unknown snapshot version")

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
		return nil, ErrUnknownSnapshotVersion
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

func NewInMemoryAdapter() *InMemoryAdapter {
	return &InMemoryAdapter{
		store: map[string][]byte{},
	}
}
