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
                               total_rewards_paid,
                               referrer_rewards_generated,
                               referees_discount_applied,
                               volume_discount_applied,
                               vega_time
	) values ($1,$2,$3,$4,$5,$6,$7,$8)`,
		stats.MarketID,
		stats.AssetID,
		stats.EpochSeq,
		stats.TotalRewardsPaid,
		stats.ReferrerRewardsGenerated,
		stats.RefereesDiscountApplied,
		stats.VolumeDiscountApplied,
		stats.VegaTime,
	)
	return err
}

func (rfs *FeesStats) GetFeesStats(ctx context.Context, marketID *entities.MarketID, assetID *entities.AssetID,
	epochSeq *uint64, referrerID, refereeID *string) (
	*entities.FeesStats, error,
) {
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

	if partyFilter := getPartyFilter(referrerID, refereeID); partyFilter != "" {
		where = append(where, partyFilter)
	}

	if len(where) > 0 {
		for i, w := range where {
			if i == 0 {
				query = fmt.Sprintf("%s WHERE %s", query, w)
				continue
			}
			query = fmt.Sprintf("%s AND %s", query, w)
		}
	}

	if epochSeq == nil { // we want the most recent stat so order and limit the query
		query = fmt.Sprintf("%s ORDER BY epoch_seq DESC LIMIT 1", query)
	}

	if err = pgxscan.Select(ctx, rfs.Connection, &stats, query, args...); err != nil {
		return nil, err
	}

	if len(stats) == 0 {
		return nil, errors.New("no referral fee stats found")
	}

	// the query returns the full JSON object and doesn't filter for the referrer/referee,
	// it only matches on the records where the json object contains the referrer/referee
	if referrerID != nil {
		// filter the results to only include the results for the given referrer
		stats[0].TotalRewardsPaid = filterPartyAmounts(stats[0].TotalRewardsPaid, *referrerID)
		stats[0].ReferrerRewardsGenerated = filterReferrerRewardsGenerated(stats[0].ReferrerRewardsGenerated, *referrerID)
	}

	if refereeID != nil {
		// filter the results to only include the results for the given referee
		for i, rrg := range stats[0].ReferrerRewardsGenerated {
			stats[0].ReferrerRewardsGenerated[i].GeneratedReward = filterPartyAmounts(rrg.GeneratedReward, *refereeID)
		}
		stats[0].RefereesDiscountApplied = filterPartyAmounts(stats[0].RefereesDiscountApplied, *refereeID)
		stats[0].VolumeDiscountApplied = filterPartyAmounts(stats[0].VolumeDiscountApplied, *refereeID)
	}

	return &stats[0], err
}

func filterPartyAmounts(totalRewardsPaid []*eventspb.PartyAmount, party string) []*eventspb.PartyAmount {
	filteredTotalRewardsPaid := make([]*eventspb.PartyAmount, 0)
	for _, reward := range totalRewardsPaid {
		if strings.EqualFold(reward.Party, party) {
			filteredTotalRewardsPaid = append(filteredTotalRewardsPaid, reward)
		}
	}
	return filteredTotalRewardsPaid
}

func filterReferrerRewardsGenerated(rewardsGenerated []*eventspb.ReferrerRewardsGenerated, referrer string) []*eventspb.ReferrerRewardsGenerated {
	filteredReferrerRewardsGenerated := make([]*eventspb.ReferrerRewardsGenerated, 0)
	for _, reward := range rewardsGenerated {
		if strings.EqualFold(reward.Referrer, referrer) {
			filteredReferrerRewardsGenerated = append(filteredReferrerRewardsGenerated, reward)
		}
	}
	return filteredReferrerRewardsGenerated
}

func getPartyFilter(referrerID, refereeID *string) string {
	builder := strings.Builder{}
	if referrerID == nil && refereeID == nil {
		return ""
	}

	builder.WriteString("(")

	if referrerID != nil {
		builder.WriteString(fmt.Sprintf(
			`total_rewards_paid @> '[{"party_id":"%s"}]'`, *referrerID,
		))
		builder.WriteString(" OR ")
		builder.WriteString(fmt.Sprintf(
			`referrer_rewards_generated @> '[{"referrer":"%s"}]'`, *referrerID,
		))
	}

	if refereeID != nil {
		if referrerID != nil {
			builder.WriteString(" OR ")
		}
		builder.WriteString(fmt.Sprintf(
			`referrer_rewards_generated @> '[{"generated_reward":[{"party":"%s"}]}]'`, *refereeID,
		))
		builder.WriteString(" OR ")
		builder.WriteString(fmt.Sprintf(
			`referees_discount_applied @> '[{"party_id":"%s"}]'`, *refereeID,
		))
		builder.WriteString(" OR ")
		builder.WriteString(fmt.Sprintf(
			`volume_discount_applied @> '[{"party_id":"%s"}]'`, *refereeID,
		))
	}

	builder.WriteString(")")

	return builder.String()
}
