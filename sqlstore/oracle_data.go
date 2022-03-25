package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type OracleData struct {
	*SQLStore
}

const (
	sqlOracleDataColumns = `public_keys, data, matched_spec_ids, broadcast_at, vega_time`
)

func NewOracleData(sqlStore *SQLStore) *OracleData {
	return &OracleData{
		SQLStore: sqlStore,
	}
}

func (od *OracleData) Add(data *entities.OracleData) error {
	ctx, cancel := context.WithTimeout(context.Background(), od.conf.Timeout.Duration)
	defer cancel()

	query := fmt.Sprintf("insert into oracle_data(%s) values ($1, $2, $3, $4, $5)", sqlOracleDataColumns)

	if _, err := od.pool.Exec(ctx, query, data.PublicKeys, data.Data, data.MatchedSpecIds, data.BroadcastAt, data.VegaTime); err != nil {
		err = fmt.Errorf("could not insert oracle data into database: %w", err)
		return err
	}

	return nil
}

func (od *OracleData) GetOracleDataBySpecID(ctx context.Context, id string, pagination entities.Pagination) ([]entities.OracleData, error) {
	specID := entities.NewSpecID(id)
	var bindVars []interface{}

	query := fmt.Sprintf(`select %s
	from oracle_data where %s = ANY(matched_spec_ids)`, sqlOracleDataColumns, nextBindVar(&bindVars, specID))

	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)
	var oracleData []entities.OracleData

	err := pgxscan.Select(ctx, od.pool, &oracleData, query, bindVars...)

	return oracleData, err
}
