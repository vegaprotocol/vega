package snapshot

import (
	cometbftdb "github.com/cometbft/cometbft-db"
)

type InMemoryDatabase struct {
	*cometbftdb.MemDB
}

func (d *InMemoryDatabase) Clear() error {
	d.MemDB = cometbftdb.NewMemDB()
	return nil
}

func NewInMemoryDatabase() *InMemoryDatabase {
	return &InMemoryDatabase{
		MemDB: cometbftdb.NewMemDB(),
	}
}
