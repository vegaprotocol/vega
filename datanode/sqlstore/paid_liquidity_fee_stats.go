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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

type PaidLiquidityFeesStats struct {
	*ConnectionSource
}

func NewPaidLiquidityFeesStats(src *ConnectionSource) *PaidLiquidityFeesStats {
	return &PaidLiquidityFeesStats{
		ConnectionSource: src,
	}
}

func (rfs *PaidLiquidityFeesStats) Add(ctx context.Context, stats *entities.PaidLiquidityFeesStats) error {
	defer metrics.StartSQLQuery("PaidLiquidityFeesStats", "Add")()
	// It's possible that a market closes in the same block as an end of epoch event.
	// In this case, the market close event will cause a paid liquidity fee stats event to be sent
	// as well as the epoch end event. In this case we just want to ignore the second event.
	_, err := rfs.Exec(
		ctx,
		`INSERT INTO paid_liquidity_fees(
			market_id,
			asset_id,
			epoch_seq,
			total_fees_paid,
			fees_paid_per_party,
			vega_time
		) values ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`,
		stats.MarketID,
		stats.AssetID,
		stats.EpochSeq,
		stats.TotalFeesPaid,
		stats.FeesPerParty,
		stats.VegaTime,
	)
	return err
}

func (lfs *PaidLiquidityFeesStats) List(
	ctx context.Context,
	marketID *entities.MarketID,
	assetID *entities.AssetID,
	epochSeq *uint64,
	partyIDs []string,
	pagination entities.CursorPagination,
	epochFrom, epochTo *uint64,
) ([]entities.PaidLiquidityFeesStats, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("PaidLiquidityFeesStats", "List")()
	var (
		args     []interface{}
		pageInfo entities.PageInfo
	)

	query := `SELECT t.market_id, t.asset_id, t.epoch_seq, t.total_fees_paid, array_to_json(array_agg(j)) as fees_per_party
	FROM paid_liquidity_fees t, jsonb_array_elements(t.fees_paid_per_party) j`

	whereClauses := []string{}

	if (marketID == nil || assetID == nil) && epochSeq == nil && epochFrom == nil && epochTo == nil {
		whereClauses = append(whereClauses, "epoch_seq = (SELECT MAX(epoch_seq) FROM paid_liquidity_fees)")
	}

	// to from range set, but wrong way around
	if epochFrom != nil && epochTo != nil && *epochFrom > *epochTo {
		epochTo, epochFrom = epochFrom, epochTo
	}
	if epochFrom != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("epoch_seq >= %s", nextBindVar(&args, *epochFrom)))
	}
	if epochTo != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("epoch_seq <= %s", nextBindVar(&args, *epochTo)))
	}
	// @TODO remove precise epoch sequence?
	if epochFrom == nil && epochTo == nil && epochSeq != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("epoch_seq = %s", nextBindVar(&args, *epochSeq)))
	}

	if marketID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("market_id = %s", nextBindVar(&args, marketID)))
	}

	if assetID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("asset_id = %s", nextBindVar(&args, assetID)))
	}

	if len(partyIDs) > 0 {
		parties := strings.Builder{}
		for i, party := range partyIDs {
			if i > 0 {
				parties.WriteString(",")
			}
			parties.WriteString(nextBindVar(&args, party))
		}

		whereClauses = append(whereClauses, fmt.Sprintf("j->>'party' IN (%s)", parties.String()))
	}

	var whereStr string
	if len(whereClauses) > 0 {
		whereStr = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	groupByStr := "GROUP BY market_id, asset_id, epoch_seq, vega_time"

	query = fmt.Sprintf("%s %s %s", query, whereStr, groupByStr)

	stats := []entities.PaidLiquidityFeesStats{}

	query, args, err := PaginateQuery[entities.PaidLiquidityFeesStatsCursor](
		query, args, paidLiquidityFeesStatsCursorOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, lfs.ConnectionSource, &stats, query, args...); err != nil {
		return nil, pageInfo, err
	}

	stats, pageInfo = entities.PageEntities[*v2.PaidLiquidityFeesEdge](stats, pagination)

	return stats, pageInfo, nil
}
