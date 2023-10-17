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

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/metrics"

	"code.vegaprotocol.io/vega/datanode/entities"
)

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
	_, err := rfs.Connection.Exec(
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
	)
	return err
}

func (rfs *FeesStats) GetFeesStats(ctx context.Context, marketID *entities.MarketID, assetID *entities.AssetID, epochSeq *uint64, partyID *string) (*entities.FeesStats, error) {
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

	if epochSeq == nil { // we want the most recent stat so order and limit the query
		where = append(where, "epoch_seq = (SELECT MAX(epoch_seq) FROM fees_stats)")
	}

	if partyFilter := getPartyFilter(partyID); partyFilter != "" {
		where = append(where, partyFilter)
	}

	if len(where) > 0 {
		query = fmt.Sprintf("%s where %s", query, strings.Join(where, " and "))
	}

	query = fmt.Sprintf("%s order by market_id, asset_id, epoch_seq desc", query)

	if err = pgxscan.Select(ctx, rfs.Connection, &stats, query, args...); err != nil {
		return nil, err
	}

	if len(stats) == 0 {
		return nil, errors.New("no  fees stats found")
	}

	// The query returns the full JSON object and doesn't filter for the party,
	// it only matches on the records where the json object contains the party.
	if partyID != nil {
		stats[0].TotalRewardsReceived = filterPartyAmounts(stats[0].TotalRewardsReceived, *partyID)
		stats[0].ReferrerRewardsGenerated = filterReferrerRewardsGenerated(stats[0].ReferrerRewardsGenerated, *partyID)
		stats[0].TotalMakerFeesReceived = filterPartyAmounts(stats[0].TotalMakerFeesReceived, *partyID)
		stats[0].MakerFeesGenerated = filterMakerFeesGenerated(stats[0].MakerFeesGenerated, *partyID)
		stats[0].RefereesDiscountApplied = filterPartyAmounts(stats[0].RefereesDiscountApplied, *partyID)
		stats[0].VolumeDiscountApplied = filterPartyAmounts(stats[0].VolumeDiscountApplied, *partyID)
	}

	return &stats[0], err
}

func filterPartyAmounts(totalRewardsReceived []*eventspb.PartyAmount, party string) []*eventspb.PartyAmount {
	filteredEntries := make([]*eventspb.PartyAmount, 0)
	for _, reward := range totalRewardsReceived {
		if strings.EqualFold(reward.Party, party) {
			filteredEntries = append(filteredEntries, reward)
		}
	}
	return filteredEntries
}

func filterReferrerRewardsGenerated(rewardsGenerated []*eventspb.ReferrerRewardsGenerated, partyID string) []*eventspb.ReferrerRewardsGenerated {
	filteredEntries := make([]*eventspb.ReferrerRewardsGenerated, 0)
	for _, reward := range rewardsGenerated {
		if strings.EqualFold(reward.Referrer, partyID) {
			filteredEntries = append(filteredEntries, reward)
		}
	}
	return filteredEntries
}

func filterMakerFeesGenerated(makerFeesGenerated []*eventspb.MakerFeesGenerated, partyID string) []*eventspb.MakerFeesGenerated {
	filteredEntries := make([]*eventspb.MakerFeesGenerated, 0)
	for _, reward := range makerFeesGenerated {
		if strings.EqualFold(reward.Taker, partyID) {
			filteredEntries = append(filteredEntries, reward)
		}
	}
	return filteredEntries
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
