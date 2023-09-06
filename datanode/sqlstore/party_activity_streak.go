package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"github.com/georgysavva/scany/pgxscan"
)

type PartyActivityStreaks struct {
	*ConnectionSource
}

func NewPartyActivityStreaks(connectionSource *ConnectionSource) *PartyActivityStreaks {
	return &PartyActivityStreaks{
		ConnectionSource: connectionSource,
	}
}

func (pas *PartyActivityStreaks) Add(
	ctx context.Context,
	activityStreak *entities.PartyActivityStreak,
) error {
	defer metrics.StartSQLQuery("PartyActivityStreaks", "Add")()

	_, err := pas.Connection.Exec(
		ctx, partyActivityStreakAddQuery, activityStreak.Fields()...)

	return err
}

func (pas *PartyActivityStreaks) Get(
	ctx context.Context,
	party entities.PartyID,
	epoch *uint64,
) (*entities.PartyActivityStreak, error) {
	defer metrics.StartSQLQuery("PartyActivityStreaks", "Get")()

	var (
		query          string
		args           []interface{}
		activityStreak *entities.PartyActivityStreak
	)
	if epoch != nil {
		query = fmt.Sprintf(
			"SELECT * FROM party_activity_streaks where party_id = %s AND epoch = %s",
			nextBindVar(&args, party), nextBindVar(&args, *epoch),
		)
	} else {
		query = fmt.Sprintf(
			"SELECT * FROM party_activity_streaks where party_id = %s ORDER BY epoch DESC LIMIT 1",
			nextBindVar(&args, party),
		)
	}

	err := pgxscan.Select(ctx, pas.Connection, &activityStreak, query, args...)

	return activityStreak, err
}

const (
	partyActivityStreakAddQuery = `INSERT INTO party_activity_streaks (party_id, active_for, inactive_for, is_active, reward_distribution_activity_multiplier, reward_vesting_activity_multiplier, epoch, traded_volume, open_volume, vega_time, tx_hash) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
`
)
