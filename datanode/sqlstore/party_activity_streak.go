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
		activityStreak []*entities.PartyActivityStreak
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
	if err != nil {
		return nil, err
	}

	if len(activityStreak) <= 0 {
		return nil, entities.ErrNotFound
	}

	return activityStreak[0], nil
}

const (
	partyActivityStreakAddQuery = `INSERT INTO party_activity_streaks (party_id, active_for, inactive_for, is_active, reward_distribution_activity_multiplier, reward_vesting_activity_multiplier, epoch, traded_volume, open_volume, vega_time, tx_hash) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
`
)
