package sqlstore

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type NetworkParameters struct {
	*SQLStore
}

func NewNetworkParameters(sqlStore *SQLStore) *NetworkParameters {
	p := &NetworkParameters{
		SQLStore: sqlStore,
	}
	return p
}

func (ps *NetworkParameters) Add(ctx context.Context, r entities.NetworkParameter) error {
	_, err := ps.pool.Exec(ctx,
		`INSERT INTO network_parameters(
			key,
			value,
			vega_time)
		 VALUES ($1,  $2,  $3)
		 ON CONFLICT (key, vega_time) DO UPDATE SET
			value = EXCLUDED.value;
		 `,
		r.Key, r.Value, r.VegaTime)
	return err
}

func (np *NetworkParameters) GetAll(ctx context.Context) ([]entities.NetworkParameter, error) {
	var nps []entities.NetworkParameter
	query := `SELECT DISTINCT ON (key) * FROM network_parameters ORDER BY key, vega_time DESC`
	err := pgxscan.Select(ctx, np.pool, &nps, query)
	return nps, err
}
