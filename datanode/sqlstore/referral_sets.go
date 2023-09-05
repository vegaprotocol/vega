package sqlstore

import (
	"context"
	"fmt"

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
