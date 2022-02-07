package sqlstore

import (
	"context"
	"encoding/hex"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Parties struct {
	*SqlStore
}

func NewParties(sqlStore *SqlStore) *Parties {
	a := &Parties{
		SqlStore: sqlStore,
	}
	return a
}

func (ps *Parties) Add(p entities.Party) error {
	ctx := context.Background()
	_, err := ps.pool.Exec(ctx,
		`INSERT INTO parties(id, vega_time)
		 VALUES ($1, $2)`,
		p.ID,
		p.VegaTime)
	return err
}

func (ps *Parties) GetByID(id string) (entities.Party, error) {
	a := entities.Party{}
	idBytes, err := hex.DecodeString(id)
	if err != nil {
		return a, ErrBadID
	}

	ctx := context.Background()
	err = pgxscan.Get(ctx, ps.pool, &a,
		`SELECT id, vega_time
		 FROM parties WHERE id=$1`,
		idBytes)
	return a, err
}

func (ps *Parties) GetAll() ([]entities.Party, error) {
	ctx := context.Background()
	parties := []entities.Party{}
	err := pgxscan.Select(ctx, ps.pool, &parties, `
		SELECT id, vega_time
		FROM parties`)
	return parties, err
}
