package sqlstore

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

var ErrPartyNotFound = errors.New("party not found")
var ErrInvalidPartyID = errors.New("invalid hex id")

type Parties struct {
	*SQLStore
}

func NewParties(sqlStore *SQLStore) *Parties {
	ps := &Parties{
		SQLStore: sqlStore,
	}
	return ps
}

// Initialise adds the built-in 'network' party which is never explicitly sent on the event
// bus, but nonetheless is necessary.
func (ps *Parties) Initialise() {
	_, err := ps.pool.Exec(context.Background(),
		`INSERT INTO parties(id) VALUES ($1) ON CONFLICT (id) DO NOTHING`,
		entities.NewPartyID("network"))
	if err != nil {
		panic(fmt.Errorf("Unable to add built-in network party: %w", err))
	}
}

func (ps *Parties) Add(ctx context.Context, p entities.Party) error {
	_, err := ps.pool.Exec(ctx,
		`INSERT INTO parties(id, vega_time)
		 VALUES ($1, $2)
		 ON CONFLICT (id) DO NOTHING`,
		p.ID,
		p.VegaTime)
	return err
}

func (ps *Parties) GetByID(ctx context.Context, id string) (entities.Party, error) {
	a := entities.Party{}
	err := pgxscan.Get(ctx, ps.pool, &a,
		`SELECT id, vega_time
		 FROM parties WHERE id=$1`,
		entities.NewOrderID(id))

	if pgxscan.NotFound(err) {
		return a, fmt.Errorf("'%v': %w", id, ErrPartyNotFound)
	}

	if errors.Is(err, entities.ErrInvalidID) {
		return a, fmt.Errorf("'%v': %w", id, ErrInvalidPartyID)
	}

	return a, err
}

func (ps *Parties) GetAll(ctx context.Context) ([]entities.Party, error) {
	parties := []entities.Party{}
	err := pgxscan.Select(ctx, ps.pool, &parties, `
		SELECT id, vega_time
		FROM parties`)
	return parties, err
}
