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

type ReferralSets struct {
	*ConnectionSource
}

var (
	referralSetOrdering = TableOrdering{
		ColumnOrdering{Name: "created_at", Sorting: ASC},
	}

	referralSetRefereeOrdering = TableOrdering{
		ColumnOrdering{Name: "joined_at", Sorting: ASC},
		ColumnOrdering{Name: "referee", Sorting: ASC},
	}

	referralSetStatsOrdering = TableOrdering{
		ColumnOrdering{Name: "at_epoch", Sorting: DESC},
		ColumnOrdering{Name: "set_id", Sorting: ASC},
		ColumnOrdering{Name: "party_id", Sorting: ASC, Ref: "referee_stats->>'party_id'"},
	}

	paidLiquidityFeesStatsCursorOrdering = TableOrdering{
		ColumnOrdering{Name: "market_id", Sorting: ASC},
		ColumnOrdering{Name: "asset_id", Sorting: ASC},
		ColumnOrdering{Name: "epoch_seq", Sorting: DESC},
	}
)

func NewReferralSets(connectionSource *ConnectionSource) *ReferralSets {
	return &ReferralSets{
		ConnectionSource: connectionSource,
	}
}

func (rs *ReferralSets) AddReferralSet(ctx context.Context, referralSet *entities.ReferralSet) error {
	defer metrics.StartSQLQuery("ReferralSets", "AddReferralSet")()
	_, err := rs.Exec(
		ctx,
		"INSERT INTO referral_sets(id, referrer, created_at, updated_at, vega_time) VALUES ($1, $2, $3, $4, $5)",
		referralSet.ID,
		referralSet.Referrer,
		referralSet.CreatedAt,
		referralSet.UpdatedAt,
		referralSet.VegaTime,
	)

	return err
}

func (rs *ReferralSets) RefereeJoinedReferralSet(ctx context.Context, referee *entities.ReferralSetReferee) error {
	defer metrics.StartSQLQuery("ReferralSets", "AddReferralSetReferee")()
	_, err := rs.Exec(
		ctx,
		"INSERT INTO referral_set_referees(referral_set_id, referee, joined_at, at_epoch, vega_time) VALUES ($1, $2, $3, $4, $5)",
		referee.ReferralSetID,
		referee.Referee,
		referee.JoinedAt,
		referee.AtEpoch,
		referee.VegaTime,
	)

	return err
}

func (rs *ReferralSets) ListReferralSets(ctx context.Context, referralSetID *entities.ReferralSetID, referrer, referee *entities.PartyID, pagination entities.CursorPagination) (
	[]entities.ReferralSet, entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("ReferralSets", "ListReferralSets")()
	var (
		sets     []entities.ReferralSet
		args     []interface{}
		err      error
		pageInfo entities.PageInfo
	)

	query := `WITH
  referees_stats AS (
    SELECT referral_set_id, COUNT(DISTINCT referee) AS total_referees
    FROM current_referral_set_referees
    GROUP BY
      referral_set_id
  )
SELECT referral_sets.*, COALESCE(referees_stats.total_referees, 0) + 1 AS total_members -- plus the referrer
FROM referral_sets
  LEFT JOIN referees_stats ON referral_sets.id = referees_stats.referral_set_id`

	// we only allow one of the following to be used as the filter
	if referralSetID != nil {
		query = fmt.Sprintf("%s WHERE referral_sets.id = %s", query, nextBindVar(&args, referralSetID))
	} else if referrer != nil {
		query = fmt.Sprintf("%s WHERE referral_sets.referrer = %s", query, nextBindVar(&args, referrer))
	} else if referee != nil {
		query = fmt.Sprintf("%s INNER JOIN current_referral_set_referees ON current_referral_set_referees.referee = %s AND referral_sets.id = current_referral_set_referees.referral_set_id", query, nextBindVar(&args, referee))
	}

	query, args, err = PaginateQuery[entities.ReferralSetCursor](query, args, referralSetOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, rs.ConnectionSource, &sets, query, args...); err != nil {
		return nil, pageInfo, err
	}

	sets, pageInfo = entities.PageEntities[*v2.ReferralSetEdge](sets, pagination)
	return sets, pageInfo, nil
}

func (rs *ReferralSets) AddReferralSetStats(ctx context.Context, stats *entities.ReferralSetStats) error {
	defer metrics.StartSQLQuery("ReferralSets", "AddReferralSetStats")()

	// Just to ensure "nil" doesn't get inserted, in place of an empty array.
	refereesStats := stats.RefereesStats
	if refereesStats == nil {
		refereesStats = []*eventspb.RefereeStats{}
	}

	_, err := rs.Exec(
		ctx,
		`INSERT INTO referral_set_stats(
			   set_id,
			   at_epoch,
			   was_eligible,
			   referral_set_running_notional_taker_volume,
			   referrer_taker_volume,
			   referees_stats,
			   vega_time,
			   reward_factors,
			   rewards_multiplier,
			   rewards_factors_multiplier)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		stats.SetID,
		stats.AtEpoch,
		stats.WasEligible,
		stats.ReferralSetRunningNotionalTakerVolume,
		stats.ReferrerTakerVolume,
		refereesStats,
		stats.VegaTime,
		stats.RewardFactors,
		stats.RewardsMultiplier,
		stats.RewardsFactorsMultiplier,
	)

	return err
}

func (rs *ReferralSets) GetReferralSetStats(ctx context.Context, setID *entities.ReferralSetID, atEpoch *uint64, referee *entities.PartyID, pagination entities.CursorPagination) ([]entities.FlattenReferralSetStats, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("ReferralSets", "GetReferralSetStats")()
	var (
		query    string
		args     []interface{}
		pageInfo entities.PageInfo
	)

	stats := []struct {
		SetID                                 entities.ReferralSetID
		AtEpoch                               uint64
		WasEligible                           bool
		ReferralSetRunningNotionalTakerVolume string
		ReferrerTakerVolume                   string
		VegaTime                              time.Time
		PartyID                               string
		DiscountFactors                       string
		EpochNotionalTakerVolume              string
		RewardFactors                         *vega.RewardFactors
		RewardsMultiplier                     string
		RewardsFactorsMultiplier              *vega.RewardFactors
	}{}

	query = `SELECT set_id,
					at_epoch,
					was_eligible,
       				vega_time,
       				referral_set_running_notional_taker_volume,
       				referrer_taker_volume,
       				reward_factors,
       				referee_stats->>'party_id' AS party_id,
       				referee_stats->>'discount_factors' AS discount_factors,
       				referee_stats->>'epoch_notional_taker_volume' AS epoch_notional_taker_volume,
					rewards_multiplier,
    				rewards_factors_multiplier
			  FROM referral_set_stats, JSONB_ARRAY_ELEMENTS(referees_stats) AS referee_stats`

	whereClauses := []string{}

	if (setID == nil || referee == nil) && atEpoch == nil {
		whereClauses = append(whereClauses, "at_epoch = (SELECT MAX(at_epoch) FROM referral_set_stats)")
	}

	if atEpoch != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("at_epoch = %s", nextBindVar(&args, *atEpoch)))
	}

	if referee != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("referee_stats->>'party_id' = %s", nextBindVar(&args, referee.String())))
	}

	if setID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("set_id = %s", nextBindVar(&args, setID)))
	}

	var whereStr string
	if len(whereClauses) > 0 {
		whereStr = " where " + strings.Join(whereClauses, " AND ")
	}

	query = fmt.Sprintf("%s %s", query, whereStr)

	query, args, err := PaginateQuery[entities.ReferralSetStatsCursor](query, args, referralSetStatsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, rs.ConnectionSource, &stats, query, args...); err != nil {
		return nil, pageInfo, err
	}

	flattenStats := []entities.FlattenReferralSetStats{}
	for _, stat := range stats {
		discountFactors := &vega.DiscountFactors{}
		if err := json.Unmarshal([]byte(stat.DiscountFactors), discountFactors); err != nil {
			return nil, pageInfo, err
		}

		flattenStats = append(flattenStats, entities.FlattenReferralSetStats{
			SetID:                                 stat.SetID,
			AtEpoch:                               stat.AtEpoch,
			WasEligible:                           stat.WasEligible,
			ReferralSetRunningNotionalTakerVolume: stat.ReferralSetRunningNotionalTakerVolume,
			ReferrerTakerVolume:                   stat.ReferrerTakerVolume,
			VegaTime:                              stat.VegaTime,
			PartyID:                               stat.PartyID,
			DiscountFactors:                       discountFactors,
			EpochNotionalTakerVolume:              stat.EpochNotionalTakerVolume,
			RewardFactors:                         stat.RewardFactors,
			RewardsMultiplier:                     stat.RewardsMultiplier,
			RewardsFactorsMultiplier:              stat.RewardsFactorsMultiplier,
		})
	}

	flattenStats, pageInfo = entities.PageEntities[*v2.ReferralSetStatsEdge](flattenStats, pagination)

	return flattenStats, pageInfo, nil
}

func (rs *ReferralSets) ListReferralSetReferees(ctx context.Context, referralSetID *entities.ReferralSetID, referrer, referee *entities.PartyID,
	pagination entities.CursorPagination, aggregationEpochs uint32) (
	[]entities.ReferralSetRefereeStats, entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("ReferralSets", "ListReferralSetReferees")()
	var (
		referees []entities.ReferralSetRefereeStats
		args     []interface{}
		err      error
		pageInfo entities.PageInfo
	)

	query := getSelectQuery(aggregationEpochs)

	var hasWhere bool
	// we only allow one of the following to be used as the filter
	if referralSetID != nil {
		query = fmt.Sprintf("%s where rf.referral_set_id = %s", query, nextBindVar(&args, referralSetID))
		hasWhere = true
	} else if referrer != nil {
		query = fmt.Sprintf("%s where rs.referrer = %s", query, nextBindVar(&args, referrer))
		hasWhere = true
	} else if referee != nil {
		query = fmt.Sprintf("%s where rf.referee = %s", query, nextBindVar(&args, referee))
		hasWhere = true
	}

	paginate := PaginateQueryWithWhere[entities.ReferralSetRefereeCursor]
	if hasWhere {
		paginate = PaginateQuery[entities.ReferralSetRefereeCursor]
	}

	query, args, err = paginate(query, args, referralSetRefereeOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, rs.ConnectionSource, &referees, query, args...); err != nil {
		return nil, pageInfo, err
	}

	referees, pageInfo = entities.PageEntities[*v2.ReferralSetRefereeEdge](referees, pagination)

	return referees, pageInfo, nil
}

func getSelectQuery(aggregationEpochs uint32) string {
	return fmt.Sprintf(`
with epoch_range as (select GREATEST(max(id) - %d, 0) as start_epoch, GREATEST(max(id), 0) as end_epoch
                     from epochs
                     where end_time is not null
), ref_period_volume (party, period_volume) as (
    select decode(ref_stats->>'party_id', 'hex'), sum((ref_stats->>'epoch_notional_taker_volume')::numeric) as period_volume
    from referral_set_stats, jsonb_array_elements(referees_stats) as ref_stats, epoch_range
    where at_epoch > epoch_range.start_epoch and at_epoch <= epoch_range.end_epoch
    and   jsonb_typeof(referees_stats) != 'null'
    group by ref_stats->>'party_id'
), ref_period_rewards (party, period_rewards) as (
    select decode(gen_rewards->>'party', 'hex'), sum((gen_rewards ->> 'quantum_amount')::numeric) as period_rewards
    from fees_stats,
         jsonb_array_elements(referrer_rewards_generated) as ref_rewards,
         jsonb_array_elements(ref_rewards->'generated_reward') as gen_rewards,
	     epoch_range
    where epoch_seq > epoch_range.start_epoch and epoch_seq <= epoch_range.end_epoch
    and jsonb_typeof(referrer_rewards_generated) != 'null'
    group by gen_rewards->>'party'
)
SELECT rf.referral_set_id, rf.referee, rf.joined_at, rf.at_epoch, rf.vega_time, coalesce(pv.period_volume, 0) period_volume, coalesce(pr.period_rewards, 0) period_rewards_paid
from current_referral_set_referees rf
join referral_sets rs on rf.referral_set_id = rs.id
left join ref_period_volume pv on rf.referee = pv.party
left join ref_period_rewards pr on rf.referee = pr.party
	`, aggregationEpochs)
}
