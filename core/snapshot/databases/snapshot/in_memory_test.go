package snapshot_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/snapshot/databases/snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryDatabase(t *testing.T) {
	t.Run("Saving and loading snapshots succeeds", testInMemoryDatabaseSavingAndLoadingSnapshotsSucceeds)
}

func testInMemoryDatabaseSavingAndLoadingSnapshotsSucceeds(t *testing.T) {
	db := snapshot.NewInMemoryDatabase()
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
