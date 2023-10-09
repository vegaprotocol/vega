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

package tree

import (
	"errors"
	"fmt"

	metadatadb "code.vegaprotocol.io/vega/core/snapshot/databases/metadata"
	snapshotdb "code.vegaprotocol.io/vega/core/snapshot/databases/snapshot"
	"code.vegaprotocol.io/vega/paths"
)

var (
	ErrDatabasesAreAlreadyInitialized    = errors.New("the databases are already initialized")
	ErrMinimumNumberOfSnapshotsToKeepIs1 = errors.New("the minimum number of snapshots to keep is 1")
)

type Options func(t *Tree) error

func WithMaxNumberOfSnapshotsToKeep(max uint64) Options {
	return func(t *Tree) error {
		if max < 1 {
			return ErrMinimumNumberOfSnapshotsToKeepIs1
		}
		t.maxNumberOfSnapshotsToKeep = max
		return nil
	}
}

func StartingAtBlockHeight(blockHeight uint64) Options {
	return func(t *Tree) error {
		t.blockHeightToStartFrom = blockHeight
		return nil
	}
}

func WithLevelDBDatabase(vegaPaths paths.Paths) Options {
	return func(t *Tree) error {
		if t.snapshotDB != nil || t.metadataDB != nil {
			return ErrDatabasesAreAlreadyInitialized
		}

		snapshotsDB, err := snapshotdb.NewLevelDBDatabase(vegaPaths)
		if err != nil {
			return fmt.Errorf("could not initialize snapshot database: %w", err)
		}
		t.snapshotDB = snapshotsDB

		metadataDB, err := metadatadb.NewLevelDBDatabase(vegaPaths)
		if err != nil {
			return fmt.Errorf("could not initialize metadata database: %w", err)
		}
		t.metadataDB = metadataDB

		return nil
	}
}

func WithInMemoryDatabase() Options {
	return func(t *Tree) error {
		t.snapshotDB = snapshotdb.NewInMemoryDatabase()
		t.metadataDB = metadatadb.NewInMemoryDatabase()
		return nil
	}
}
