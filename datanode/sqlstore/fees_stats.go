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
	"errors"
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/georgysavva/scany/pgxscan"
	"golang.org/x/exp/maps"
)

var feesStatsByPartyColumn = []string{
	"market_id",
	"asset_id",
	"party_id",
	"epoch_seq",
	"total_rewards_received",
	"referees_discount_applied",
	"volume_discount_applied",
	"total_maker_fees_received",
	"vega_time",
}

type FeesStats struct {
	*ConnectionSource
}

func NewFeesStats(src *ConnectionSource) *FeesStats {
	return &FeesStats{
		ConnectionSource: src,
	}
}

func (rfs *FeesStats) AddFeesStats(ctx context.Context, stats *entities.FeesStats) error {
	defer metrics.StartSQLQuery("FeesStats", "AddFeesStats")()

	if _, err := rfs.Exec(
		ctx,
		`INSERT INTO fees_stats(
			   market_id,
			   asset_id,
			   epoch_seq,
			   total_rewards_received,
			   referrer_rewards_generated,
			   referees_discount_applied,
			   volume_discount_applied,
			   total_maker_fees_received,
			   maker_fees_generated,
			   vega_time
	         ) values ($1,$2,$3,$4,$5,$6,$7,$8, $9, $10)`,
		stats.MarketID,
		stats.AssetID,
		stats.EpochSeq,
		stats.TotalRewardsReceived,
		stats.ReferrerRewardsGenerated,
		stats.RefereesDiscountApplied,
		stats.VolumeDiscountApplied,
		stats.TotalMakerFeesReceived,
		stats.MakerFeesGenerated,
		stats.VegaTime,
	); err != nil {
		return fmt.Errorf("could not execute insertion in `fees_stats`: %w", err)
	}

	batcher := NewListBatcher[*feesStatsForPartyRow]("fees_stats_by_party", feesStatsByPartyColumn)
	partiesStats := computePartiesStats(stats)
	for _, s := range partiesStats {
		batcher.Add(s)
	}
	if _, err := batcher.Flush(ctx, rfs.ConnectionSource); err != nil {
		return err
	}

	return nil
}

func (rfs *FeesStats) StatsForParty(ctx context.Context, partyID entities.PartyID, assetID *entities.AssetID, fromEpoch, toEpoch *uint64) ([]entities.FeesStatsForParty, error) {
	defer metrics.StartSQLQuery("FeesStats", "StatsForParty")()

	var args []interface{}

	where := []string{
		fmt.Sprintf("party_id = %s", nextBindVar(&args, partyID)),
	}

	if assetID != nil {
		where = append(where, fmt.Sprintf("asset_id = %s", nextBindVar(&args, *assetID)))
	}

	if fromEpoch == nil && toEpoch == nil {
		where = append(where, "epoch_seq = (SELECT MAX(epoch_seq) FROM fees_stats)")
	}
	if fromEpoch != nil {
		where = append(where, fmt.Sprintf("epoch_seq >= %s", nextBindVar(&args, *fromEpoch)))
	}
	if toEpoch != nil {
		where = append(where, fmt.Sprintf("epoch_seq <= %s", nextBindVar(&args, *toEpoch)))
	}

	query := fmt.Sprintf(`select
            asset_id,
            sum(total_maker_fees_received) as total_maker_fees_received,
            sum(referees_discount_applied) as referees_discount_applied,
            sum(total_rewards_received) as total_rewards_received,
            sum(volume_discount_applied) as volume_discount_applied
        from fees_stats_by_party where %s group by party_id, asset_id order by asset_id`,
		strings.Join(where, " and "),
	)

	var rows []feesStatsForPartyRow
	if err := pgxscan.Select(ctx, rfs.ConnectionSource, &rows, query, args...); err != nil {
		return nil, err
	}

	stats := make([]entities.FeesStatsForParty, 0, len(rows))
	for _, row := range rows {
		stats = append(stats, entities.FeesStatsForParty{
			AssetID:                 row.AssetID,
			TotalRewardsReceived:    row.TotalRewardsReceived.String(),
			RefereesDiscountApplied: row.RefereesDiscountApplied.String(),
			VolumeDiscountApplied:   row.VolumeDiscountApplied.String(),
			TotalMakerFeesReceived:  row.TotalMakerFeesReceived.String(),
		})
	}

	return stats, nil
}

func (rfs *FeesStats) GetFeesStats(ctx context.Context, marketID *entities.MarketID, assetID *entities.AssetID, epochSeq *uint64, partyIDs []string, epochFrom, epochTo *uint64) (*entities.FeesStats, error) {
	defer metrics.StartSQLQuery("FeesStats", "GetFeesStats")()
	var (
		stats []entities.FeesStats
		err   error
		args  []interface{}
	)

	if marketID != nil && assetID != nil {
		return nil, errors.New("only a marketID or assetID should be provided")
	}

	query := `SELECT * FROM fees_stats`
	where := make([]string, 0)

	if epochSeq != nil {
		where = append(where, fmt.Sprintf("epoch_seq = %s", nextBindVar(&args, *epochSeq)))
	}

	if assetID != nil {
		where = append(where, fmt.Sprintf("asset_id = %s", nextBindVar(&args, *assetID)))
	}

	if marketID != nil {
		where = append(where, fmt.Sprintf("market_id = %s", nextBindVar(&args, *marketID)))
	}

	if epochFrom != nil && epochTo != nil && *epochFrom > *epochTo {
		epochFrom, epochTo = epochTo, epochFrom
	}
	if epochFrom != nil {
		where = append(where, fmt.Sprintf("epoch_seq >= %s", nextBindVar(&args, *epochFrom)))
		epochSeq = nil
	}
	if epochTo != nil {
		where = append(where, fmt.Sprintf("epoch_seq <= %s", nextBindVar(&args, *epochTo)))
		epochSeq = nil
	}

	if epochSeq == nil && epochFrom == nil && epochTo == nil { // we want the most recent stat so order and limit the query
		where = append(where, "epoch_seq = (SELECT MAX(epoch_seq) FROM fees_stats)")
	}

	if partyFilter := getPartiesFilter(partyIDs); partyFilter != "" {
		where = append(where, partyFilter)
	}

	if len(where) > 0 {
		query = fmt.Sprintf("%s where %s", query, strings.Join(where, " and "))
	}

	query = fmt.Sprintf("%s order by market_id, asset_id, epoch_seq desc", query)

	if err = pgxscan.Select(ctx, rfs.ConnectionSource, &stats, query, args...); err != nil {
		return nil, err
	}

	if len(stats) == 0 {
		return nil, errors.New("no  fees stats found")
	}

	// The query returns the full JSON object and doesn't filter for the party,
	// it only matches on the records where the json object contains the party.
	if len(partyIDs) > 0 {
		stats[0].TotalRewardsReceived = filterPartyAmounts(stats[0].TotalRewardsReceived, partyIDs...)
		stats[0].ReferrerRewardsGenerated = filterReferrerRewardsGenerated(stats[0].ReferrerRewardsGenerated, partyIDs...)
		stats[0].TotalMakerFeesReceived = filterPartyAmounts(stats[0].TotalMakerFeesReceived, partyIDs...)
		stats[0].MakerFeesGenerated = filterMakerFeesGenerated(stats[0].MakerFeesGenerated, partyIDs...)
		stats[0].RefereesDiscountApplied = filterPartyAmounts(stats[0].RefereesDiscountApplied, partyIDs...)
		stats[0].VolumeDiscountApplied = filterPartyAmounts(stats[0].VolumeDiscountApplied, partyIDs...)
	}

	return &stats[0], err
}

func filterPartyAmounts(totalRewardsReceived []*eventspb.PartyAmount, parties ...string) []*eventspb.PartyAmount {
	filteredEntries := make([]*eventspb.PartyAmount, 0)
	for _, reward := range totalRewardsReceived {
		for _, party := range parties {
			if strings.EqualFold(reward.Party, party) {
				filteredEntries = append(filteredEntries, reward)
			}
		}
	}
	return filteredEntries
}

func filterReferrerRewardsGenerated(rewardsGenerated []*eventspb.ReferrerRewardsGenerated, parties ...string) []*eventspb.ReferrerRewardsGenerated {
	filteredEntries := make([]*eventspb.ReferrerRewardsGenerated, 0)
	for _, reward := range rewardsGenerated {
		for _, partyID := range parties {
			if strings.EqualFold(reward.Referrer, partyID) {
				filteredEntries = append(filteredEntries, reward)
			}
		}
	}
	return filteredEntries
}

func filterMakerFeesGenerated(makerFeesGenerated []*eventspb.MakerFeesGenerated, parties ...string) []*eventspb.MakerFeesGenerated {
	filteredEntries := make([]*eventspb.MakerFeesGenerated, 0)
	for _, reward := range makerFeesGenerated {
		for _, partyID := range parties {
			if strings.EqualFold(reward.Taker, partyID) {
				filteredEntries = append(filteredEntries, reward)
			}
		}
	}
	return filteredEntries
}

func getPartiesFilter(parties []string) string {
	if len(parties) == 0 {
		return ""
	}
	parts := make([]string, 0, len(parties))
	for _, id := range parties {
		parts = append(parts, getPartyFilter(&id))
	}
	return fmt.Sprintf("(%s)", strings.Join(parts, " OR "))
}

func getPartyFilter(partyID *string) string {
	builder := strings.Builder{}
	if partyID == nil {
		return ""
	}

	builder.WriteString("(")

	builder.WriteString(fmt.Sprintf(
		`total_rewards_received @> '[{"party_id":"%s"}]'`, *partyID,
	))
	builder.WriteString(" OR ")
	builder.WriteString(fmt.Sprintf(
		`referrer_rewards_generated @> '[{"referrer":"%s"}]'`, *partyID,
	))
	builder.WriteString(" OR ")
	builder.WriteString(fmt.Sprintf(
		`referees_discount_applied @> '[{"party_id":"%s"}]'`, *partyID,
	))
	builder.WriteString(" OR ")
	builder.WriteString(fmt.Sprintf(
		`volume_discount_applied @> '[{"party_id":"%s"}]'`, *partyID,
	))
	builder.WriteString(" OR ")
	builder.WriteString(fmt.Sprintf(
		`total_maker_fees_received @> '[{"party_id":"%s"}]'`, *partyID,
	))
	builder.WriteString(" OR ")
	builder.WriteString(fmt.Sprintf(
		`maker_fees_generated @> '[{"taker":"%s"}]'`, *partyID,
	))

	builder.WriteString(")")

	return builder.String()
}

func computePartiesStats(stats *entities.FeesStats) []*feesStatsForPartyRow {
	partiesStats := map[string]*feesStatsForPartyRow{}

	for _, t := range stats.TotalMakerFeesReceived {
		partyStats := ensurePartyStats(stats, partiesStats, t)
		partyStats.TotalMakerFeesReceived = partyStats.TotalMakerFeesReceived.Add(num.MustDecimalFromString(t.Amount))
	}

	for _, t := range stats.TotalRewardsReceived {
		partyStats := ensurePartyStats(stats, partiesStats, t)
		partyStats.TotalRewardsReceived = partyStats.TotalRewardsReceived.Add(num.MustDecimalFromString(t.Amount))
	}

	for _, t := range stats.VolumeDiscountApplied {
		partyStats := ensurePartyStats(stats, partiesStats, t)
		partyStats.VolumeDiscountApplied = partyStats.VolumeDiscountApplied.Add(num.MustDecimalFromString(t.Amount))
	}

	for _, t := range stats.RefereesDiscountApplied {
		partyStats := ensurePartyStats(stats, partiesStats, t)
		partyStats.RefereesDiscountApplied = partyStats.RefereesDiscountApplied.Add(num.MustDecimalFromString(t.Amount))
	}

	return maps.Values(partiesStats)
}

func ensurePartyStats(stats *entities.FeesStats, partiesStats map[string]*feesStatsForPartyRow, t *eventspb.PartyAmount) *feesStatsForPartyRow {
	partyStats, ok := partiesStats[t.Party]
	if !ok {
		partyStats = &feesStatsForPartyRow{
			MarketID:                stats.MarketID,
			AssetID:                 stats.AssetID,
			PartyID:                 entities.PartyID(t.Party),
			EpochSeq:                stats.EpochSeq,
			TotalRewardsReceived:    num.DecimalZero(),
			RefereesDiscountApplied: num.DecimalZero(),
			VolumeDiscountApplied:   num.DecimalZero(),
			TotalMakerFeesReceived:  num.DecimalZero(),
			VegaTime:                stats.VegaTime,
		}
		partiesStats[t.Party] = partyStats
	}
	return partyStats
}

type feesStatsForPartyRow struct {
	MarketID                entities.MarketID
	AssetID                 entities.AssetID
	PartyID                 entities.PartyID
	EpochSeq                uint64
	TotalRewardsReceived    num.Decimal
	RefereesDiscountApplied num.Decimal
	VolumeDiscountApplied   num.Decimal
	TotalMakerFeesReceived  num.Decimal
	VegaTime                time.Time
}

func (f feesStatsForPartyRow) ToRow() []interface{} {
	return []any{
		f.MarketID,
		f.AssetID,
		f.PartyID,
		f.EpochSeq,
		f.TotalRewardsReceived,
		f.RefereesDiscountApplied,
		f.VolumeDiscountApplied,
		f.TotalMakerFeesReceived,
		f.VegaTime,
	}
}
