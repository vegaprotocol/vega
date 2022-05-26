package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	"github.com/georgysavva/scany/pgxscan"
)

type OracleData struct {
	*ConnectionSource
}

const (
	sqlOracleDataColumns = `public_keys, data, matched_spec_ids, broadcast_at, vega_time`
)

func NewOracleData(connectionSource *ConnectionSource) *OracleData {
	return &OracleData{
		ConnectionSource: connectionSource,
	}
}

func (od *OracleData) Add(ctx context.Context, data *entities.OracleData) error {
	defer metrics.StartSQLQuery("OracleData", "Add")()
	query := fmt.Sprintf("insert into oracle_data(%s) values ($1, $2, $3, $4, $5)", sqlOracleDataColumns)

	if _, err := od.Connection.Exec(ctx, query, data.PublicKeys, data.Data, data.MatchedSpecIds, data.BroadcastAt, data.VegaTime); err != nil {
		err = fmt.Errorf("could not insert oracle data into database: %w", err)
		return err
	}

	return nil
}

func (od *OracleData) GetOracleDataBySpecID(ctx context.Context, id string, pagination entities.OffsetPagination) ([]entities.OracleData, error) {
	specID := entities.NewSpecID(id)
	var bindVars []interface{}

	query := fmt.Sprintf(`select %s
	from oracle_data where %s = ANY(matched_spec_ids)`, sqlOracleDataColumns, nextBindVar(&bindVars, specID))

	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)
	var oracleData []entities.OracleData

	defer metrics.StartSQLQuery("OracleData", "GetBySpecID")()
	err := pgxscan.Select(ctx, od.Connection, &oracleData, query, bindVars...)

	return oracleData, err
}

func (od *OracleData) ListOracleData(ctx context.Context, pagination entities.OffsetPagination) ([]entities.OracleData, error) {
	var data []entities.OracleData
	query := fmt.Sprintf(`select distinct on (id) %s
from oracle_data
order by id, vega_time desc`, sqlOracleDataColumns)

	var bindVars []interface{}
	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)
	defer metrics.StartSQLQuery("OracleData", "ListOracleData")()
	err := pgxscan.Select(ctx, od.Connection, &data, query, bindVars...)
	return data, err
}
