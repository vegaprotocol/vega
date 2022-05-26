package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	"github.com/georgysavva/scany/pgxscan"
)

type OracleSpec struct {
	*ConnectionSource
}

const (
	sqlOracleSpecColumns = `id, created_at, updated_at, public_keys, filters, status, vega_time`
)

func NewOracleSpec(connectionSource *ConnectionSource) *OracleSpec {
	return &OracleSpec{
		ConnectionSource: connectionSource,
	}
}

func (os *OracleSpec) Upsert(ctx context.Context, spec *entities.OracleSpec) error {
	query := fmt.Sprintf(`insert into oracle_specs(%s)
values ($1, $2, $3, $4, $5, $6, $7)
on conflict (id, vega_time) do update
set
	created_at=EXCLUDED.created_at,
	updated_at=EXCLUDED.updated_at,
	public_keys=EXCLUDED.public_keys,
	filters=EXCLUDED.filters,
	status=EXCLUDED.status`, sqlOracleSpecColumns)

	defer metrics.StartSQLQuery("OracleSpec", "Upsert")()
	if _, err := os.Connection.Exec(ctx, query, spec.ID, spec.CreatedAt, spec.UpdatedAt, spec.PublicKeys,
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

	defer metrics.StartSQLQuery("OracleSpec", "GetByID")()
	err := pgxscan.Get(ctx, os.Connection, &spec, query, entities.NewSpecID(specID))
	return spec, err
}

func (os *OracleSpec) GetSpecs(ctx context.Context, pagination entities.OffsetPagination) ([]entities.OracleSpec, error) {
	var specs []entities.OracleSpec
	query := fmt.Sprintf(`select distinct on (id) %s
from oracle_specs
order by id, vega_time desc`, sqlOracleSpecColumns)

	var bindVars []interface{}
	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)
	defer metrics.StartSQLQuery("OracleSpec", "ListOracleSpecs")()
	err := pgxscan.Select(ctx, os.Connection, &specs, query, bindVars...)
	return specs, err
}
