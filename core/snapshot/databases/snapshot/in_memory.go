package snapshot

import (
	db "github.com/tendermint/tm-db"
)

type InMemoryDatabase struct {
	*db.MemDB
}

func (d *InMemoryDatabase) Clear() error {
	d.MemDB = db.NewMemDB()
	return nil
}

func NewInMemoryDatabase() *InMemoryDatabase {
	return &InMemoryDatabase{
		MemDB: db.NewMemDB(),
	}
}
