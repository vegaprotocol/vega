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

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
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
	_, err := rs.Connection.Exec(
		ctx,
		"INSERT INTO referral_sets(id, referrer, created_at, updated_at, vega_time) values ($1, $2, $3, $4, $5)",
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
	_, err := rs.Connection.Exec(
		ctx,
		"INSERT INTO referral_set_referees(referral_set_id, referee, joined_at, at_epoch, vega_time) values ($1, $2, $3, $4, $5)",
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

	query := `SELECT DISTINCT rs.id as id, rs.referrer as referrer, rs.created_at as created_at, rs.updated_at as updated_at, rs.vega_time as vega_time
			  FROM referral_sets rs
			  LEFT JOIN referral_set_referees r on rs.id = r.referral_set_id` // LEFT JOIN because a referral set may not have any referees joined yet.

	// we only allow one of the following to be used as the filter
	if referralSetID != nil {
		query = fmt.Sprintf("%s where rs.id = %s", query, nextBindVar(&args, referralSetID))
	} else if referrer != nil {
		query = fmt.Sprintf("%s where rs.referrer = %s", query, nextBindVar(&args, referrer))
	} else if referee != nil {
		query = fmt.Sprintf("%s where r.referee = %s", query, nextBindVar(&args, referee))
	}

	query, args, err = PaginateQuery[entities.ReferralSetCursor](query, args, referralSetOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, rs.Connection, &sets, query, args...); err != nil {
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

	_, err := rs.Connection.Exec(
		ctx,
		`INSERT INTO referral_set_stats(
			   set_id,
			   at_epoch,
			   was_eligible,
			   referral_set_running_notional_taker_volume,
			   referrer_taker_volume,
			   referees_stats,
			   vega_time,
			   reward_factor,
			   rewards_multiplier,
			   rewards_factor_multiplier)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		stats.SetID,
		stats.AtEpoch,
		stats.WasEligible,
		stats.ReferralSetRunningNotionalTakerVolume,
		stats.ReferrerTakerVolume,
		refereesStats,
		stats.VegaTime,
		stats.RewardFactor,
		stats.RewardsMultiplier,
		stats.RewardsFactorMultiplier,
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

	query = `SELECT set_id,
					at_epoch,
					was_eligible,
       				vega_time,
       				referral_set_running_notional_taker_volume,
       				referrer_taker_volume,
       				reward_factor,
       				referee_stats->>'party_id' as party_id,
       				referee_stats->>'discount_factor' as discount_factor,
       				referee_stats->>'epoch_notional_taker_volume' as epoch_notional_taker_volume,
					rewards_multiplier,
    				rewards_factor_multiplier
			  FROM referral_set_stats, jsonb_array_elements(referees_stats) AS referee_stats`

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

	stats := []entities.FlattenReferralSetStats{}

	query, args, err := PaginateQuery[entities.ReferralSetStatsCursor](query, args, referralSetStatsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, rs.Connection, &stats, query, args...); err != nil {
		return nil, pageInfo, err
	}

	stats, pageInfo = entities.PageEntities[*v2.ReferralSetStatsEdge](stats, pagination)

	return stats, pageInfo, nil
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

	if err := pgxscan.Select(ctx, rs.Connection, &referees, query, args...); err != nil {
		return nil, pageInfo, err
	}

	referees, pageInfo = entities.PageEntities[*v2.ReferralSetRefereeEdge](referees, pagination)

	return referees, pageInfo, nil
}

func getSelectQuery(aggregationEpochs uint32) string {
	return fmt.Sprintf(`
with epoch_range as (select coalesce(max(id) - %d, 0) as start_epoch, coalesce(max(id), 0) as end_epoch
                     from epochs
                     where end_time is not null
), ref_period_volume (party, period_volume) as (
    select decode(ref_stats->>'party_id', 'hex'), sum((ref_stats->>'epoch_notional_taker_volume')::numeric) as period_volume
    from referral_set_stats, jsonb_array_elements(referees_stats) as ref_stats, epoch_range
    where at_epoch >= epoch_range.start_epoch and at_epoch <= epoch_range.end_epoch
    and   jsonb_typeof(referees_stats) != 'null'
    group by ref_stats->>'party_id'
), ref_period_rewards (party, period_rewards) as (
    select decode(gen_rewards->>'party', 'hex'), sum((gen_rewards ->> 'quantum_amount')::numeric) as period_rewards
    from fees_stats,
         jsonb_array_elements(referrer_rewards_generated) as ref_rewards,
         jsonb_array_elements(ref_rewards->'generated_reward') as gen_rewards,
	     epoch_range
    where epoch_seq >= epoch_range.start_epoch and epoch_seq <= epoch_range.end_epoch
    and jsonb_typeof(referrer_rewards_generated) != 'null'
    group by gen_rewards->>'party'
)
SELECT rf.referral_set_id, rf.referee, rf.joined_at, rf.at_epoch, rf.vega_time, coalesce(pv.period_volume, 0) period_volume, coalesce(pr.period_rewards, 0) period_rewards_paid
from referral_set_referees rf
join referral_sets rs on rf.referral_set_id = rs.id
left join ref_period_volume pv on rf.referee = pv.party
left join ref_period_rewards pr on rf.referee = pr.party
	`, aggregationEpochs)
}
