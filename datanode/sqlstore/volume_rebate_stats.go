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

var volumeRebateStatsOrdering = TableOrdering{
	ColumnOrdering{Name: "at_epoch", Sorting: DESC},
	ColumnOrdering{Name: "party_id", Sorting: ASC, Ref: "stats->>'party_id'"},
}

type (
	VolumeRebateStats struct {
		*ConnectionSource
	}
)

func NewVolumeRebateStats(connectionSource *ConnectionSource) *VolumeRebateStats {
	return &VolumeRebateStats{
		ConnectionSource: connectionSource,
	}
}

func (s *VolumeRebateStats) Add(ctx context.Context, stats *entities.VolumeRebateStats) error {
	defer metrics.StartSQLQuery("VolumeRebateStats", "Add")()
	_, err := s.Exec(
		ctx,
		`INSERT INTO volume_rebate_stats(at_epoch, parties_volume_rebate_stats, vega_time)
			values ($1, $2, $3)`,
		stats.AtEpoch,
		stats.PartiesVolumeRebateStats,
		stats.VegaTime,
	)

	return err
}

func (s *VolumeRebateStats) Stats(ctx context.Context, atEpoch *uint64, partyID *string, pagination entities.CursorPagination) ([]entities.FlattenVolumeRebateStats, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("VolumeRebateStats", "VolumeRebateStats")()

	var (
		args     []any
		pageInfo entities.PageInfo
	)

	filters := []string{}
	filters = append(filters, "jsonb_typeof(parties_volume_rebate_stats) != 'null'")

	if atEpoch != nil {
		filters = append(filters, fmt.Sprintf("at_epoch = %s", nextBindVar(&args, atEpoch)))
	}
	if partyID != nil {
		filters = append(filters, fmt.Sprintf("stats->>'party_id' = %s", nextBindVar(&args, partyID)))
	}

	if partyID == nil && atEpoch == nil {
		filters = append(filters, "at_epoch = (SELECT MAX(at_epoch) FROM volume_rebate_stats)")
	}

	stats := []entities.FlattenVolumeRebateStats{}
	query := `select at_epoch, stats->>'party_id' as party_id, stats->>'maker_volume_fraction' as maker_volume_fraction, stats->>'additional_rebate' as additional_rebate, stats->>'maker_fees_received' as maker_fees_received, vega_time from volume_rebate_stats, jsonb_array_elements(parties_volume_rebate_stats) AS stats`

	if len(filters) > 0 {
		query = fmt.Sprintf("%s where %s", query, strings.Join(filters, " and "))
	}

	query, args, err := PaginateQuery[entities.VolumeRebateStatsCursor](query, args, volumeRebateStatsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, s.ConnectionSource, &stats, query, args...); err != nil {
		return nil, pageInfo, err
	}

	stats, pageInfo = entities.PageEntities[*v2.VolumeRebateStatsEdge](stats, pagination)

	return stats, pageInfo, nil
}
