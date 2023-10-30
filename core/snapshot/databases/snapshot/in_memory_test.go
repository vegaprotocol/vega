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
