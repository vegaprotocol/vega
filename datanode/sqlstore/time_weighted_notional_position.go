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

	"github.com/georgysavva/scany/pgxscan"
)

type (
	TimeWeightedNotionalPosition struct {
		*ConnectionSource
	}
)

func NewTimeWeightedNotionalPosition(connectionSource *ConnectionSource) *TimeWeightedNotionalPosition {
	return &TimeWeightedNotionalPosition{
		ConnectionSource: connectionSource,
	}
}

func (tw *TimeWeightedNotionalPosition) Upsert(ctx context.Context, twNotionalPos entities.TimeWeightedNotionalPosition) error {
	_, err := tw.Connection.Exec(ctx, `
		INSERT INTO time_weighted_notional_positions (asset_id, party_id, epoch_seq, time_weighted_notional_position, last_updated)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (asset_id, party_id, epoch_seq)
		DO UPDATE
			SET time_weighted_notional_position = $4,
			last_updated = $5
	`,
		twNotionalPos.AssetID, twNotionalPos.PartyID, twNotionalPos.EpochSeq,
		twNotionalPos.TimeWeightedNotionalPosition, twNotionalPos.LastUpdated)
	return err
}

func (tw *TimeWeightedNotionalPosition) Get(ctx context.Context, assetID entities.AssetID, partyID entities.PartyID, epochSeq *uint64) (entities.TimeWeightedNotionalPosition, error) {
	var twNotionalPos entities.TimeWeightedNotionalPosition
	if epochSeq == nil {
		err := pgxscan.Get(ctx, tw.Connection, &twNotionalPos,
			`SELECT * FROM time_weighted_notional_positions WHERE asset_id = $1 AND party_id = $2 ORDER BY epoch_seq DESC LIMIT 1`,
			assetID, partyID)
		if err != nil {
			return twNotionalPos, err
		}
		return twNotionalPos, nil
	}
	err := pgxscan.Get(ctx, tw.Connection, &twNotionalPos,
		`SELECT * FROM time_weighted_notional_positions WHERE asset_id = $1 AND party_id = $2 AND epoch_seq = $3`,
		assetID, partyID, *epochSeq)
	if err != nil {
		return twNotionalPos, err
	}
	return twNotionalPos, err
}
