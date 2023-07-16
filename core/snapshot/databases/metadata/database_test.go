package metadata_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/snapshot/databases/metadata"
	"code.vegaprotocol.io/vega/paths"
	tmtypes "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabase(t *testing.T) {
	t.Run("Saving and loading snapshot metadata succeeds", testSavingAndLoadingSnapshotMetadataSucceeds)
}

func testSavingAndLoadingSnapshotMetadataSucceeds(t *testing.T) {
	tcs := []struct {
		name string
		// Used to instantiate only one connection at a time of the test.
		newDatabaseFn func(t *testing.T) *metadata.Database
	}{
		{
			name: "with LevelDB adapter",
			newDatabaseFn: func(t *testing.T) *metadata.Database {
				t.Helper()
				return newLevelDBDatabase(t)
			},
		}, {
			name: "with in-memory adapter",
			newDatabaseFn: func(t *testing.T) *metadata.Database {
				t.Helper()
				return metadata.NewDatabase(metadata.NewInMemoryAdapter())
			},
		},
	}

	snapshotV1 := &tmtypes.Snapshot{Height: 1, Format: 2, Chunks: 7, Hash: []byte{1, 2}, Metadata: []byte{1}}
	snapshotV2 := &tmtypes.Snapshot{Height: 2, Format: 2, Chunks: 7, Hash: []byte{2, 2}, Metadata: []byte{2}}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			db := tc.newDatabaseFn(tt)
			defer func() {
				// Ensure closing does not have problem of any kind.
				require.NoError(tt, db.Close())
			}()

			// Loading 2 snapshots to verify they are properly saved, and do not
			// override each other.
			require.NoError(tt, db.Save(1, snapshotV1))
			require.NoError(tt, db.Save(2, snapshotV2))

			// Verify both snapshot can be retrieve and match the originals.
			loadedSnapshotV1, err := db.Load(1)
			require.NoError(tt, err)
			assert.Equal(tt, snapshotV1, loadedSnapshotV1)

			loadedSnapshotV2, err := db.Load(2)
			require.NoError(tt, err)
			assert.Equal(tt, snapshotV2, loadedSnapshotV2)

			// Removing the snapshots from the database.
			require.NoError(tt, db.Clear())

			// Verify both snapshot can no longer be retrieved from the database.
			loadedSnapshotV1AfterClear, err := db.Load(1)
			assert.Error(tt, err)
			assert.Nil(tt, loadedSnapshotV1AfterClear)

			loadedSnapshotV2AfterClear, err := db.Load(2)
			assert.Error(tt, err)
			assert.Nil(tt, loadedSnapshotV2AfterClear)
		})
	}
}

func newLevelDBDatabase(t *testing.T) *metadata.Database {
	t.Helper()

	vegaHome := paths.New(t.TempDir())
	adapter, err := metadata.NewLevelDBAdapter(vegaHome)
	require.NoError(t, err)
	return metadata.NewDatabase(adapter)
}
