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

package sqlstore_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeWeightedNotionalPosition_Upsert(t *testing.T) {
	ctx := tempTransaction(t)
	t.Run("Upsert should create a time weighted notional position record if it dpesn't exist", func(t *testing.T) {
		tw := sqlstore.NewTimeWeightedNotionalPosition(connectionSource)
		want := entities.TimeWeightedNotionalPosition{
			AssetID:                      entities.AssetID(GenerateID()),
			PartyID:                      entities.PartyID(GenerateID()),
			GameID:                       entities.GameID(GenerateID()),
			EpochSeq:                     1,
			TimeWeightedNotionalPosition: 1000,
			VegaTime:                     time.Now().Truncate(time.Microsecond),
		}
		err := tw.Upsert(ctx, want)
		require.NoError(t, err)
		var got entities.TimeWeightedNotionalPosition
		err = pgxscan.Get(ctx, connectionSource.Connection, &got,
			`SELECT * FROM time_weighted_notional_positions WHERE asset_id = $1 AND party_id = $2 and game_id = $3 and epoch_seq = $4`,
			want.AssetID, want.PartyID, want.GameID, want.EpochSeq)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("Upsert should update a time weighted notional position record if it exists", func(t *testing.T) {
		tw := sqlstore.NewTimeWeightedNotionalPosition(connectionSource)
		want := entities.TimeWeightedNotionalPosition{
			AssetID:                      entities.AssetID(GenerateID()),
			PartyID:                      entities.PartyID(GenerateID()),
			GameID:                       entities.GameID(GenerateID()),
			EpochSeq:                     2,
			TimeWeightedNotionalPosition: 1000,
			VegaTime:                     time.Now().Truncate(time.Microsecond),
		}
		err := tw.Upsert(ctx, want)
		require.NoError(t, err)
		want.TimeWeightedNotionalPosition = 2000
		err = tw.Upsert(ctx, want)
		require.NoError(t, err)
		var got entities.TimeWeightedNotionalPosition
		err = pgxscan.Get(ctx, connectionSource.Connection, &got,
			`SELECT * FROM time_weighted_notional_positions WHERE asset_id = $1 AND party_id = $2 and game_id = $3 and epoch_seq = $4`,
			want.AssetID, want.PartyID, want.GameID, want.EpochSeq)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

func TestTimeWeightedNotionalPosition_Get(t *testing.T) {
	ctx := tempTransaction(t)
	t.Run("Get should return a time weighted notional position record if it exists", func(t *testing.T) {
		tw := sqlstore.NewTimeWeightedNotionalPosition(connectionSource)
		want := entities.TimeWeightedNotionalPosition{
			AssetID:                      entities.AssetID(GenerateID()),
			PartyID:                      entities.PartyID(GenerateID()),
			GameID:                       entities.GameID(GenerateID()),
			EpochSeq:                     1,
			TimeWeightedNotionalPosition: 1000,
			VegaTime:                     time.Now().Truncate(time.Microsecond),
		}
		err := tw.Upsert(ctx, want)
		require.NoError(t, err)
		got, err := tw.Get(ctx, want.AssetID, want.PartyID, want.GameID, ptr.From(want.EpochSeq))
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("Get should return the latest time weighted notional position record if no epoch is specified", func(t *testing.T) {
		tw := sqlstore.NewTimeWeightedNotionalPosition(connectionSource)
		want := entities.TimeWeightedNotionalPosition{
			AssetID:                      entities.AssetID(GenerateID()),
			PartyID:                      entities.PartyID(GenerateID()),
			GameID:                       entities.GameID(GenerateID()),
			EpochSeq:                     1,
			TimeWeightedNotionalPosition: 1000,
			VegaTime:                     time.Now().Truncate(time.Microsecond),
		}
		err := tw.Upsert(ctx, want)
		require.NoError(t, err)
		want.EpochSeq = 2
		want.TimeWeightedNotionalPosition = 2000
		want.VegaTime = want.VegaTime.Add(time.Second)
		err = tw.Upsert(ctx, want)
		require.NoError(t, err)
		got, err := tw.Get(ctx, want.AssetID, want.PartyID, want.GameID, nil)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}
