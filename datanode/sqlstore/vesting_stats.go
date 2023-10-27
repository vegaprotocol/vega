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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"github.com/georgysavva/scany/pgxscan"
)

type (
	VestingStats struct {
		*ConnectionSource
	}
)

func NewVestingStats(connectionSource *ConnectionSource) *VestingStats {
	return &VestingStats{
		ConnectionSource: connectionSource,
	}
}

func (vs *VestingStats) Add(ctx context.Context, stats *entities.VestingStatsUpdated) error {
	defer metrics.StartSQLQuery("PartyVestingStats", "Add")()

	for _, v := range stats.PartyVestingStats {
		_, err := vs.Connection.Exec(ctx,
			`INSERT INTO party_vesting_stats(party_id, at_epoch, reward_bonus_multiplier, quantum_balance, vega_time)
         VALUES ($1, $2, $3, $4, $5)
         ON CONFLICT (vega_time, party_id) DO NOTHING`,
			v.PartyID,
			stats.AtEpoch,
			v.RewardBonusMultiplier,
			v.QuantumBalance,
			stats.VegaTime,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (vs *VestingStats) GetByPartyID(
	ctx context.Context, id string,
) (entities.PartyVestingStats, error) {
	defer metrics.StartSQLQuery("Parties", "GetByID")()

	pvs := entities.PartyVestingStats{}
	err := pgxscan.Get(ctx, vs.Connection, &pvs,
		`SELECT party_id, at_epoch, reward_bonus_multiplier, quantum_balance, vega_time
		 FROM party_vesting_stats_current WHERE party_id=$1`,
		entities.PartyID(id))

	return pvs, vs.wrapE(err)
}
