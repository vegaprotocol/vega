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
	"context"
	"encoding/hex"
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"

	"github.com/stretchr/testify/require"
)

func addSnapshot(t *testing.T, ctx context.Context, ss *sqlstore.CoreSnapshotData, bs *sqlstore.Blocks, entity entities.CoreSnapshotData) {
	t.Helper()
	block := addTestBlock(t, ctx, bs)
	entity.VegaTime = block.VegaTime
	entity.BlockHash = hex.EncodeToString(block.Hash)
	entity.TxHash = generateTxHash()
	require.NoError(t, ss.Add(ctx, entity))
}

func TestGetSnapshots(t *testing.T) {
	ctx := tempTransaction(t)

	ss := sqlstore.NewCoreSnapshotData(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	addSnapshot(t, ctx, ss, bs, entities.CoreSnapshotData{BlockHeight: 100, VegaCoreVersion: "v0.65.0"})

	var rowCount int
	err := connectionSource.Connection.QueryRow(ctx, `select count(*) from core_snapshots`).Scan(&rowCount)
	require.NoError(t, err)
	require.Equal(t, 1, rowCount)

	entities, _, err := ss.List(ctx, entities.DefaultCursorPagination(true))
	require.NoError(t, err)
	require.Equal(t, 1, len(entities))
	require.Equal(t, uint64(100), entities[0].BlockHeight)
	require.Equal(t, "v0.65.0", entities[0].VegaCoreVersion)
}
