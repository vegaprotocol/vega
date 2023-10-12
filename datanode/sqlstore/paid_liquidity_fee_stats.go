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
	"strings"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/entities"
)

type PaidLiquidityFeeStats struct {
	*ConnectionSource
}

func NewPaidLiquidityFeeStats(src *ConnectionSource) *PaidLiquidityFeeStats {
	return &PaidLiquidityFeeStats{
		ConnectionSource: src,
	}
}

func (rfs *PaidLiquidityFeeStats) Add(ctx context.Context, stats *entities.PaidLiquidityFeeStats) error {
	defer metrics.StartSQLQuery("PaidLiquidityFeeStats", "Add")()
	_, err := rfs.Connection.Exec(
		ctx,
		`INSERT INTO paid_liquidity_fees(
			market_id,
			asset_id,
			epoch_seq,
			total_fees_paid,
			fees_paid_per_party
		) values ($1,$2,$3,$4,$5)`,
		stats.MarketID,
		stats.AssetID,
		stats.EpochSeq,
		stats.TotalFeesPaid,
		stats.FeesPerParty,
	)
	return err
}

func (lfs *PaidLiquidityFeeStats) List(
	ctx context.Context,
	marketID *entities.MarketID,
	assetID *entities.AssetID,
	epochSeq *uint64,
	partyIDs []string,
	pagination entities.CursorPagination,
) ([]entities.PaidLiquidityFeeStats, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("PaidLiquidityFeeStats", "List")()
	var (
		args     []interface{}
		pageInfo entities.PageInfo
	)

	query := `SELECT t.market_id, t.asset_id, t.epoch_seq, t.total_fees_paid, array_to_json(array_agg(j)) as Fees_per_party
	FROM paid_liquidity_fees t, jsonb_array_elements(t.fees_paid_per_party) j`

	whereClauses := []string{}

	if (marketID == nil || assetID == nil) && epochSeq == nil && len(partyIDs) == 0 {
		whereClauses = append(whereClauses, "epoch_seq = (SELECT MAX(epoch_seq) FROM paid_liquidity_fees)")
	}

	if epochSeq != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("epoch_seq = %s", nextBindVar(&args, *epochSeq)))
	}

	if marketID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("market_id = %s", nextBindVar(&args, marketID)))
	}

	if assetID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("asset_id = %s", nextBindVar(&args, assetID)))
	}

	if len(partyIDs) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("j->>'party' IN (%s)", nextBindVar(&args, partyIDs)))
	}

	var whereStr string
	if len(whereClauses) > 0 {
		whereStr = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	groupByStr := "GROUP BY market_id, asset_id, epoch_seq"

	query = fmt.Sprintf("%s %s %s", query, whereStr, groupByStr)

	stats := []entities.PaidLiquidityFeeStats{}

	query, args, err := PaginateQuery[entities.PaidLiquidityFeeStatsCursor](
		query, args, paidLiquidityFeeStatsCursorOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, lfs.Connection, &stats, query, args...); err != nil {
		return nil, pageInfo, err
	}

	stats, pageInfo = entities.PageEntities[*v2.PaidLiquidityFeesEdge](stats, pagination)

	return stats, pageInfo, nil
}
