// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
	sqlOracleDataColumns = `signers, data, meta_data, matched_spec_ids, broadcast_at, error, tx_hash, vega_time, seq_num`
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
	query := fmt.Sprintf("insert into oracle_data(%s) values ($1, $2, $3, $4, $5, $6, $7, $8, $9)", sqlOracleDataColumns)

	if _, err := od.Connection.Exec(
		ctx, query,
		oracleData.ExternalData.Data.Signers, oracleData.ExternalData.Data.Data, oracleData.ExternalData.Data.MetaData,
		oracleData.ExternalData.Data.MatchedSpecIds, oracleData.ExternalData.Data.BroadcastAt,
		oracleData.ExternalData.Data.Error, oracleData.ExternalData.Data.TxHash,
		oracleData.ExternalData.Data.VegaTime, oracleData.ExternalData.Data.SeqNum,
	); err != nil {
		err = fmt.Errorf("could not insert oracle data into database: %w", err)
		return err
	}
	return nil
}

func (od *OracleData) ListOracleData(ctx context.Context, id string, pagination entities.Pagination) ([]entities.OracleData, entities.PageInfo, error) {
	switch p := pagination.(type) {
	case entities.CursorPagination:
		return listOracleDataBySpecIDCursorPagination(ctx, od.Connection, id, p)
	default:
		panic("unsupported pagination")
	}
}

func (od *OracleData) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.OracleData, error) {
	defer metrics.StartSQLQuery("OracleData", "GetByTxHash")()

	var data []entities.Data
	query := fmt.Sprintf(`SELECT %s FROM oracle_data WHERE tx_hash = $1`, sqlOracleDataColumns)
	err := pgxscan.Select(ctx, od.Connection, &data, query, txHash)
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

	query := ""
	if len(id) > 0 {
		specID := entities.SpecID(id)
		query = fmt.Sprintf(`select %s
	from oracle_data where %s = ANY(matched_spec_ids)`, sqlOracleDataColumns, nextBindVar(&bindVars, specID))
	} else {
		query = fmt.Sprintf(`select %s
	from oracle_data`, sqlOracleDataColumns)
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
