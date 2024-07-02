// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

type OracleData struct {
	*ConnectionSource
}

const (
	sqlOracleDataColumns = `signers, data, meta_data, broadcast_at, error, tx_hash, vega_time, seq_num`
	oracleDataQuery      = `SELECT od.*, aggregated.spec_ids as matched_spec_ids
	FROM
		oracle_data od
	LEFT JOIN LATERAL (
		SELECT ARRAY_AGG(spec_id) AS spec_ids
		FROM oracle_data_oracle_specs ods
		WHERE od.vega_time = ods.vega_time
		AND od.seq_num = ods.seq_num
	) aggregated ON true
	`
)

var oracleDataOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "signers", Sorting: ASC},
}

func NewOracleData(connectionSource *ConnectionSource) *OracleData {
	return &OracleData{
		ConnectionSource: connectionSource,
	}
}

func (od *OracleData) Add(ctx context.Context, oracleData *entities.OracleData) error {
	defer metrics.StartSQLQuery("OracleData", "Add")()
	query := fmt.Sprintf("insert into oracle_data(%s) values ($1, $2, $3, $4, $5, $6, $7, $8)", sqlOracleDataColumns)

	if _, err := od.Exec(
		ctx, query,
		oracleData.ExternalData.Data.Signers, oracleData.ExternalData.Data.Data, oracleData.ExternalData.Data.MetaData,
		oracleData.ExternalData.Data.BroadcastAt,
		oracleData.ExternalData.Data.Error, oracleData.ExternalData.Data.TxHash,
		oracleData.ExternalData.Data.VegaTime, oracleData.ExternalData.Data.SeqNum,
	); err != nil {
		err = fmt.Errorf("could not insert oracle data into database: %w", err)
		return err
	}

	query2 := "insert into oracle_data_oracle_specs(vega_time, seq_num, spec_id) values ($1, $2, unnest($3::bytea[]))"
	if _, err := od.Exec(
		ctx, query2,
		oracleData.ExternalData.Data.VegaTime, oracleData.ExternalData.Data.SeqNum, oracleData.ExternalData.Data.MatchedSpecIds,
	); err != nil {
		err = fmt.Errorf("could not insert oracle data join into database: %w", err)
		return err
	}

	return nil
}

func (od *OracleData) ListOracleData(ctx context.Context, id string, pagination entities.Pagination) ([]entities.OracleData, entities.PageInfo, error) {
	switch p := pagination.(type) {
	case entities.CursorPagination:
		return listOracleDataBySpecIDCursorPagination(ctx, od.ConnectionSource, id, p)
	default:
		panic("unsupported pagination")
	}
}

func (od *OracleData) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.OracleData, error) {
	defer metrics.StartSQLQuery("OracleData", "GetByTxHash")()

	var data []entities.Data
	query := fmt.Sprintf(`%s WHERE tx_hash = $1`, oracleDataQuery)
	err := pgxscan.Select(ctx, od.ConnectionSource, &data, query, txHash)
	if err != nil {
		return nil, err
	}

	return scannedDataToOracleData(data), nil
}

func scannedDataToOracleData(scanned []entities.Data) []entities.OracleData {
	oracleData := []entities.OracleData{}
	if len(scanned) > 0 {
		for _, s := range scanned {
			oracleData = append(oracleData, entities.OracleData{
				ExternalData: &entities.ExternalData{
					Data: &entities.Data{
						Signers:        s.Signers,
						Data:           s.Data,
						MetaData:       s.MetaData,
						MatchedSpecIds: s.MatchedSpecIds,
						BroadcastAt:    s.BroadcastAt,
						Error:          s.Error,
						TxHash:         s.TxHash,
						VegaTime:       s.VegaTime,
						SeqNum:         s.SeqNum,
					},
				},
			})
		}
	}

	return oracleData
}

func listOracleDataBySpecIDCursorPagination(ctx context.Context, conn Connection, id string, pagination entities.CursorPagination) (
	[]entities.OracleData, entities.PageInfo, error,
) {
	var (
		oracleData []entities.OracleData
		data       = []entities.Data{}

		pageInfo entities.PageInfo
		bindVars []interface{}
		err      error
	)

	query := oracleDataQuery
	andOrWhere := "WHERE"

	if len(id) > 0 {
		specID := entities.SpecID(id)

		query = fmt.Sprintf(`%s
		WHERE EXISTS (SELECT 1 from  oracle_data_oracle_specs ods
		  WHERE  od.vega_time = ods.vega_time
		  AND    od.seq_num = ods.seq_num
		  AND    ods.spec_id=%s)`, query, nextBindVar(&bindVars, specID))

		andOrWhere = "AND"
	}

	// if the cursor is empty, we should restrict the query to the last day of data as otherwise, the query will scan the full hypertable
	// we only do this if we are returning the newest first data because that should be kept in memory by TimescaleDB anyway.
	// If we have a first N cursor traversing newest first data, without an after cursor, we should also restrict by date.
	// Traversing from the oldest data to the newest data will result in table scans and take time as we don't know what the oldest data is due to retention policies.
	// Anything after the first page will have a vega time in the cursor so this will not be needed.
	if pagination.HasForward() && !pagination.Forward.HasCursor() && pagination.NewestFirst {
		query = fmt.Sprintf("%s %s vega_time > now() - interval '1 day'", query, andOrWhere)
	}

	query, bindVars, err = PaginateQuery[entities.OracleDataCursor](query, bindVars, oracleDataOrdering, pagination)
	if err != nil {
		return oracleData, pageInfo, err
	}

	defer metrics.StartSQLQuery("OracleData", "ListOracleData")()
	// NOTE: If any error during the scan occurred, we return empty oracle data object.
	if err = pgxscan.Select(ctx, conn, &data, query, bindVars...); err != nil {
		return oracleData, pageInfo, err
	}

	oracleData = scannedDataToOracleData(data)

	oracleData, pageInfo = entities.PageEntities[*v2.OracleDataEdge](oracleData, pagination)
	return oracleData, pageInfo, nil
}
