package sqlstore

import (
	"context"
	"fmt"

	events "code.vegaprotocol.io/vega/protos/vega/events/v1"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"

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

func (rs *ReferralSets) ListReferralSets(ctx context.Context, referralSetID *entities.ReferralSetID, pagination entities.CursorPagination) (
	[]entities.ReferralSet, entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("ReferralSets", "ListReferralSets")()
	var (
		sets     []entities.ReferralSet
		args     []interface{}
		err      error
		pageInfo entities.PageInfo
	)

	query := `SELECT id, referrer, created_at, updated_at, vega_time
			  FROM referral_sets`
	if referralSetID != nil {
		query = fmt.Sprintf("%s WHERE id = %s", query, nextBindVar(&args, referralSetID))
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
	_, err := rs.Connection.Exec(
		ctx,
		`INSERT INTO referral_set_stats(set_id, at_epoch, referral_set_running_notional_taker_volume, referees_stats, vega_time)
			values ($1, $2, $3, $4, $5)`,
		stats.SetID,
		stats.AtEpoch,
		stats.ReferralSetRunningNotionalTakerVolume,
		stats.RefereesStats,
		stats.VegaTime,
	)

	return err
}

func (rs *ReferralSets) GetReferralSetStats(ctx context.Context, setID entities.ReferralSetID, atEpoch *uint64, referee *entities.PartyID) (entities.ReferralSetStats, error) {
	defer metrics.StartSQLQuery("ReferralSets", "GetReferralSetStats")()
	var (
		stats entities.ReferralSetStats
		query string
		args  []interface{}
	)

	if referee == nil {
		query = fmt.Sprintf(`SELECT set_id, at_epoch, referral_set_running_notional_taker_volume, referees_stats, vega_time
			  FROM referral_set_stats
			  WHERE set_id = %s`, nextBindVar(&args, setID))
	} else {
		query = fmt.Sprintf(`SELECT set_id, at_epoch, referral_set_running_notional_taker_volume, party_id, discount_factor, reward_factor, vega_time
		FROM referral_set_referee_stats
		WHERE set_id = %s`, nextBindVar(&args, setID))
	}

	if atEpoch != nil {
		query = fmt.Sprintf("%s AND at_epoch = %s", query, nextBindVar(&args, *atEpoch))
	}
	if referee == nil {
		// just get the last record from the stats and return all the referee stats
		query = fmt.Sprintf("%s ORDER BY at_epoch DESC LIMIT 1", query)
	} else {
		// filter the referee stats by the referee
		query = fmt.Sprintf("%s AND party_id = %s", query, nextBindVar(&args, referee.String()))
		// then get most recent epoch for that referee
		query = fmt.Sprintf("%s ORDER BY at_epoch DESC LIMIT 1", query)
	}

	if referee == nil {
		if err := pgxscan.Get(ctx, rs.Connection, &stats, query, args...); err != nil {
			return stats, err
		}
	} else {
		var refStats entities.ReferralSetRefereeStats
		if err := pgxscan.Get(ctx, rs.Connection, &refStats, query, args...); err != nil {
			return stats, err
		}

		stats.SetID = refStats.SetID
		stats.AtEpoch = refStats.AtEpoch
		stats.ReferralSetRunningNotionalTakerVolume = refStats.ReferralSetRunningNotionalTakerVolume
		stats.RefereesStats = []*events.RefereeStats{
			{
				PartyId:        refStats.PartyID,
				DiscountFactor: refStats.DiscountFactor,
				RewardFactor:   refStats.RewardFactor,
			},
		}
		stats.VegaTime = refStats.VegaTime
	}

	return stats, nil
}

func (rs *ReferralSets) ListReferralSetReferees(ctx context.Context, referralSetID entities.ReferralSetID, pagination entities.CursorPagination) (
	[]entities.ReferralSetReferee, entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("ReferralSets", "ListReferralSetReferees")()
	var (
		referees []entities.ReferralSetReferee
		args     []interface{}
		err      error
		pageInfo entities.PageInfo
	)

	query := `SELECT referral_set_id, referee, joined_at, at_epoch, vega_time from referral_set_referees where referral_set_id = $1`
	args = append(args, referralSetID)

	query, args, err = PaginateQuery[entities.ReferralSetRefereeCursor](query, args, referralSetRefereeOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, rs.Connection, &referees, query, args...); err != nil {
		return nil, pageInfo, err
	}

	referees, pageInfo = entities.PageEntities[*v2.ReferralSetRefereeEdge](referees, pagination)

	return referees, pageInfo, nil
}
