package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

var volumeDiscountStatsOrdering = TableOrdering{
	ColumnOrdering{Name: "at_epoch", Sorting: DESC},
	ColumnOrdering{Name: "party_id", Sorting: ASC},
}

type (
	VolumeDiscountStats struct {
		*ConnectionSource
	}
)

func NewVolumeDiscountStats(connectionSource *ConnectionSource) *VolumeDiscountStats {
	return &VolumeDiscountStats{
		ConnectionSource: connectionSource,
	}
}

func (s *VolumeDiscountStats) Add(ctx context.Context, stats *entities.VolumeDiscountStats) error {
	defer metrics.StartSQLQuery("VolumeDiscountStats", "Add")()
	_, err := s.Connection.Exec(
		ctx,
		`INSERT INTO volume_discount_stats(at_epoch, parties_volume_discount_stats, vega_time)
			values ($1, $2, $3)`,
		stats.AtEpoch,
		stats.PartiesVolumeDiscountStats,
		stats.VegaTime,
	)

	return err
}

func (s *VolumeDiscountStats) Stats(ctx context.Context, atEpoch *uint64, partyID *string, pagination entities.CursorPagination) ([]entities.FlattenVolumeDiscountStats, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("VolumeDiscountStats", "VolumeDiscountStats")()

	var (
		args     []any
		pageInfo entities.PageInfo
	)

	filters := []string{}
	if atEpoch != nil {
		filters = append(filters, fmt.Sprintf("at_epoch = %s", nextBindVar(&args, atEpoch)))
	}
	if partyID != nil {
		filters = append(filters, fmt.Sprintf("stats->>'party_id' = %s", nextBindVar(&args, partyID)))
	}

	if partyID == nil && atEpoch == nil {
		filters = append(filters, "at_epoch = (SELECT MAX(at_epoch) FROM volume_discount_stats)")
	}

	stats := []entities.FlattenVolumeDiscountStats{}
	query := `select at_epoch, stats->>'party_id' as party_id, stats->>'running_volume' as running_volume, stats->>'discount_factor' as discount_factor, vega_time from volume_discount_stats, jsonb_array_elements(parties_volume_discount_stats) AS stats`

	if len(filters) > 0 {
		query = fmt.Sprintf("%s where %s", query, strings.Join(filters, " and "))
	}

	query, args, err := PaginateQuery[entities.VolumeDiscountStatsCursor](query, args, volumeDiscountStatsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, s.Connection, &stats, query, args...); err != nil {
		return nil, pageInfo, err
	}

	stats, pageInfo = entities.PageEntities[*v2.VolumeDiscountStatsEdge](stats, pagination)

	return stats, pageInfo, nil
}
