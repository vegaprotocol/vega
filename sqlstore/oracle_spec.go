package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type OracleSpec struct {
	*SQLStore
}

const (
	sqlOracleSpecColumns = `id, created_at, updated_at, public_keys, filters, status, vega_time`
)

func NewOracleSpec(sqlStore *SQLStore) *OracleSpec {
	return &OracleSpec{
		SQLStore: sqlStore,
	}
}

func (os *OracleSpec) Upsert(spec *entities.OracleSpec) error {
	ctx, cancel := context.WithTimeout(context.Background(), os.conf.Timeout.Duration)
	defer cancel()

	query := fmt.Sprintf(`insert into oracle_specs(%s)
values ($1, $2, $3, $4, $5, $6, $7)
on conflict (id, vega_time) do update
set
	created_at=EXCLUDED.created_at,
	updated_at=EXCLUDED.updated_at,
	public_keys=EXCLUDED.public_keys,
	filters=EXCLUDED.filters,
	status=EXCLUDED.status`, sqlOracleSpecColumns)

	if _, err := os.pool.Exec(ctx, query, spec.ID, spec.CreatedAt, spec.UpdatedAt, spec.PublicKeys,
		spec.Filters, spec.Status, spec.VegaTime); err != nil {
		return err
	}

	return nil
}

func (os *OracleSpec) GetSpecByID(ctx context.Context, specID string) (entities.OracleSpec, error) {
	var spec entities.OracleSpec
	query := fmt.Sprintf(`select distinct on (id) %s
from oracle_specs
where id = $1
order by id, vega_time desc`, sqlOracleSpecColumns)

	err := pgxscan.Get(ctx, os.pool, &spec, query, entities.NewSpecID(specID))
	return spec, err
}

func (os *OracleSpec) GetSpecs(ctx context.Context, pagination entities.Pagination) ([]entities.OracleSpec, error) {
	var specs []entities.OracleSpec
	query := fmt.Sprintf(`select distinct on (id) %s
from oracle_specs
order by id, vega_time desc`, sqlOracleSpecColumns)

	var bindVars []interface{}
	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)
	err := pgxscan.Select(ctx, os.pool, &specs, query, bindVars...)
	return specs, err
}
