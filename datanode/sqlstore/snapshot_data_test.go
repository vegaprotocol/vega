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
