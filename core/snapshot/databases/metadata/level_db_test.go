package metadata_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/snapshot/databases/metadata"
	"code.vegaprotocol.io/vega/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmtypes "github.com/tendermint/tendermint/abci/types"
)

func TestLevelDBDatabase(t *testing.T) {
	t.Run("Saving and loading snapshot metadata succeeds", testLevelDBDatabaseSavingAndLoadingSnapshotMetadataSucceeds)
	t.Run("Finding a version by block height succeeds", testLevelDBDatabaseFindingVersionByBlockHeightSucceeds)
	t.Run("Removing a version succeeds", testLevelDBDatabaseRemovingVersionSucceeds)
}

func testLevelDBDatabaseSavingAndLoadingSnapshotMetadataSucceeds(t *testing.T) {
	db := newLevelDBDatabase(t)
	defer func() {
		// Ensure closing does not have problem of any kind.
		require.NoError(t, db.Close())
	}()

	snapshotV1 := &tmtypes.Snapshot{Height: 1, Format: 2, Chunks: 7, Hash: []byte{1, 2}, Metadata: []byte{1}}
	snapshotV2 := &tmtypes.Snapshot{Height: 2, Format: 2, Chunks: 7, Hash: []byte{2, 2}, Metadata: []byte{2}}

	// Verifying a new database is empty.
	assert.True(t, db.IsEmpty(), "the database should not contain data")

	// Saving 2 snapshots to verify they are properly saved, and do not
	// override each other.
	require.NoError(t, db.Save(1, snapshotV1))
	require.NoError(t, db.Save(2, snapshotV2))

	// Verifying the database correctly states it's not empty when not.
	assert.False(t, db.IsEmpty(), "the database should contain data")

	// Verify both snapshot can be retrieve and match the originals.
	loadedSnapshotV1, err := db.Load(1)
	require.NoError(t, err)
	assert.Equal(t, snapshotV1, loadedSnapshotV1)

	loadedSnapshotV2, err := db.Load(2)
	require.NoError(t, err)
	assert.Equal(t, snapshotV2, loadedSnapshotV2)

	// Removing the snapshots from the database.
	require.NoError(t, db.Clear())

	// Verifying the database correctly states it's empty when is.
	assert.True(t, db.IsEmpty(), "the database should not contain data")

	// Verify both snapshot can no longer be retrieved from the database.
	loadedSnapshotV1AfterClear, err := db.Load(1)
	assert.Error(t, err)
	assert.Nil(t, loadedSnapshotV1AfterClear)

	loadedSnapshotV2AfterClear, err := db.Load(2)
	assert.Error(t, err)
	assert.Nil(t, loadedSnapshotV2AfterClear)
}

func testLevelDBDatabaseFindingVersionByBlockHeightSucceeds(t *testing.T) {
	db := newLevelDBDatabase(t)
	defer func() {
		// Ensure closing does not have problem of any kind.
		require.NoError(t, db.Close())
	}()

	snapshotV1 := &tmtypes.Snapshot{Height: 1, Format: 2, Chunks: 7, Hash: []byte{1, 2}, Metadata: []byte{1}}
	snapshotV2 := &tmtypes.Snapshot{Height: 2, Format: 2, Chunks: 7, Hash: []byte{2, 2}, Metadata: []byte{2}}

	// Saving 2 snapshots.
	require.NoError(t, db.Save(1, snapshotV1))
	require.NoError(t, db.Save(2, snapshotV2))

	// Looking for a height that has no match.
	versionNotFound, err := db.FindVersionByBlockHeight(3)

	require.NoError(t, err)
	assert.Equal(t, int64(-1), versionNotFound)

	// Looking for a height that has no match.
	versionFound, err := db.FindVersionByBlockHeight(2)

	require.NoError(t, err)
	assert.Equal(t, int64(2), versionFound)
}

func testLevelDBDatabaseRemovingVersionSucceeds(t *testing.T) {
	db := newLevelDBDatabase(t)
	defer func() {
		// Ensure closing does not have problem of any kind.
		require.NoError(t, db.Close())
	}()

	snapshotV1 := &tmtypes.Snapshot{Height: 1, Format: 2, Chunks: 7, Hash: []byte{1, 2}, Metadata: []byte{1}}
	snapshotV2 := &tmtypes.Snapshot{Height: 2, Format: 2, Chunks: 7, Hash: []byte{2, 3}, Metadata: []byte{2}}
	snapshotV3 := &tmtypes.Snapshot{Height: 3, Format: 2, Chunks: 7, Hash: []byte{3, 4}, Metadata: []byte{3}}
	snapshotV4 := &tmtypes.Snapshot{Height: 4, Format: 2, Chunks: 7, Hash: []byte{4, 5}, Metadata: []byte{4}}
	snapshotV5 := &tmtypes.Snapshot{Height: 5, Format: 2, Chunks: 7, Hash: []byte{5, 6}, Metadata: []byte{5}}

	// Saving 2 snapshots.
	require.NoError(t, db.Save(1, snapshotV1))
	require.NoError(t, db.Save(2, snapshotV2))
	require.NoError(t, db.Save(3, snapshotV3))
	require.NoError(t, db.Save(4, snapshotV4))
	require.NoError(t, db.Save(5, snapshotV5))

	// Deleting first snapshot
	require.NoError(t, db.Delete(1))

	// Looking for a height that has no match.
	snapshotNotFound, err := db.Load(1)

	require.Error(t, err)
	assert.Nil(t, snapshotNotFound)

	// Deleting first snapshot
	require.NoError(t, db.DeleteRange(2, 5))

	expectedDeletion := []int64{2, 3, 4}
	for deletedVersion := range expectedDeletion {
		snapshotNotFound, err = db.Load(int64(deletedVersion))

		require.Error(t, err, "Version %d should have been deleted", deletedVersion)
		assert.Nilf(t, snapshotNotFound, "Version %d should have been deleted", deletedVersion)
	}

	// Looking for a height that has no match.
	versionFound, err := db.FindVersionByBlockHeight(5)

	require.NoError(t, err)
	assert.Equal(t, int64(5), versionFound)
}

func newLevelDBDatabase(t *testing.T) *metadata.LevelDBDatabase {
	t.Helper()

	db, err := metadata.NewLevelDBDatabase(paths.New(t.TempDir()))
	require.NoError(t, err)

	return db
}
