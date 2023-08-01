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
