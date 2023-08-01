package snapshot_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/snapshot/databases/snapshot"
	"code.vegaprotocol.io/vega/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevelDBDatabase(t *testing.T) {
	t.Run("Saving and loading snapshots succeeds", testLevelDDatabaseSavingAndLoadingSnapshotsSucceeds)
}

func testLevelDDatabaseSavingAndLoadingSnapshotsSucceeds(t *testing.T) {
	vegaHome := paths.New(t.TempDir())

	db, err := snapshot.NewLevelDBDatabase(vegaHome)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, db.Close())
	}()

	key := []byte{1, 2}
	value := []byte{3, 4}

	require.NoError(t, db.SetSync(key, value))

	returnedValue, err := db.Get(key)
	require.NoError(t, err)
	assert.Equal(t, value, returnedValue)

	require.NoError(t, db.Clear())

	returnedValueAfterClear, err := db.Get(key)
	assert.NoError(t, err)
	assert.Nil(t, returnedValueAfterClear)
}
