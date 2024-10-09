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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/georgysavva/scany/pgxscan"
)

var volumeDiscountStatsOrdering = TableOrdering{
	ColumnOrdering{Name: "at_epoch", Sorting: DESC},
	ColumnOrdering{Name: "party_id", Sorting: ASC, Ref: "stats->>'party_id'"},
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
	_, err := s.Exec(
		ctx,
		`INSERT INTO volume_discount_stats(at_epoch, parties_volume_discount_stats, vega_time)
			values ($1, $2, $3)`,
		stats.AtEpoch,
		stats.PartiesVolumeDiscountStats,
		stats.VegaTime,
	)

	return err
}

func (s *VolumeDiscountStats) LatestStats(ctx context.Context, partyID string) (entities.VolumeDiscountStats, error) {
	query := `SELECT * FROM volume_discount_stats WHERE
	at_epoch = (SELECT id - 1 from current_epochs ORDER BY id DESC, vega_time DESC FETCH FIRST ROW ONLY) AND
	EXISTS (SELECT TRUE FROM jsonb_array_elements(parties_volume_discount_stats) ps WHERE ps->>'party_id' = '$1')`
	ent := []entities.VolumeDiscountStats{}
	if err := pgxscan.Select(ctx, s.ConnectionSource, &ent, query, partyID); err != nil {
		return entities.VolumeDiscountStats{}, err
	}
	if len(ent) == 0 {
		return entities.VolumeDiscountStats{}, nil
	}
	final := ent[0]
	stats := make([]*eventspb.PartyVolumeDiscountStats, 0, len(ent))
	for _, e := range ent {
		for _, stat := range e.PartiesVolumeDiscountStats {
			if stat.PartyId == partyID {
				stats = append(stats, stat)
			}
		}
	}
	// running volume and discount factor to be used to match the volume discount program
	final.PartiesVolumeDiscountStats = stats
	return final, nil
}

func (s *VolumeDiscountStats) Stats(ctx context.Context, atEpoch *uint64, partyID *string, pagination entities.CursorPagination) ([]entities.FlattenVolumeDiscountStats, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("VolumeDiscountStats", "VolumeDiscountStats")()

	var (
		args     []any
		pageInfo entities.PageInfo
	)

	filters := []string{}
	filters = append(filters, "jsonb_typeof(parties_volume_discount_stats) != 'null'")

	if atEpoch != nil {
		filters = append(filters, fmt.Sprintf("at_epoch = %s", nextBindVar(&args, atEpoch)))
	}
	if partyID != nil {
		filters = append(filters, fmt.Sprintf("stats->>'party_id' = %s", nextBindVar(&args, partyID)))
	}

	if partyID == nil && atEpoch == nil {
		filters = append(filters, "at_epoch = (SELECT MAX(at_epoch) FROM volume_discount_stats)")
	}

	stats := []struct {
		AtEpoch         uint64
		PartyID         string
		RunningVolume   string
		DiscountFactors string
		VegaTime        time.Time
	}{}
	query := `select at_epoch, stats->>'party_id' as party_id, stats->>'running_volume' as running_volume, stats->>'discount_factors' as discount_factors, vega_time from volume_discount_stats, jsonb_array_elements(parties_volume_discount_stats) AS stats`

	if len(filters) > 0 {
		query = fmt.Sprintf("%s where %s", query, strings.Join(filters, " and "))
	}

	query, args, err := PaginateQuery[entities.VolumeDiscountStatsCursor](query, args, volumeDiscountStatsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, s.ConnectionSource, &stats, query, args...); err != nil {
		return nil, pageInfo, err
	}

	flattenStats := []entities.FlattenVolumeDiscountStats{}
	for _, stat := range stats {
		discountFactors := &vega.DiscountFactors{}
		if err := json.Unmarshal([]byte(stat.DiscountFactors), discountFactors); err != nil {
			return nil, pageInfo, err
		}

		flattenStats = append(flattenStats, entities.FlattenVolumeDiscountStats{
			AtEpoch:         stat.AtEpoch,
			PartyID:         stat.PartyID,
			DiscountFactors: discountFactors,
			RunningVolume:   stat.RunningVolume,
			VegaTime:        stat.VegaTime,
		})
	}

	flattenStats, pageInfo = entities.PageEntities[*v2.VolumeDiscountStatsEdge](flattenStats, pagination)

	return flattenStats, pageInfo, nil
}
