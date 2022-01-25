package snapshots

import (
	"fmt"

	"github.com/cosmos/iavl"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	db "github.com/tendermint/tm-db"
)

// SnapshotData is a representation of the information we an scrape from the avl tree.
type SnapshotData struct {
	Version int64  `json:"version"`
	Hash    []byte `json:"hash"`
	Size    int64  `json:"size"`
}

func SnapshotsHeightsFromTree(tree *iavl.MutableTree) (map[uint64]SnapshotData, error) {
	trees := make(map[uint64]SnapshotData)
	versions := tree.AvailableVersions()

	for _, version := range versions {
		v, err := tree.LazyLoadVersion(int64(version))
		if err != nil {
			return nil, err
		}

		app, err := types.AppStateFromTree(tree.ImmutableTree)
		if err != nil {
			fmt.Println("Failed to get app state data from snapshot",
				logging.Error(err),
				logging.Int64("snapshot-version", v),
			)
			continue
		}

		snap, err := types.SnapshotFromTree(tree.ImmutableTree)
		if err != nil {
			return nil, err
		}

		trees[app.AppState.Height] = SnapshotData{
			Version: v,
			Hash:    snap.Hash,
			Size:    tree.Size(),
		}
	}

	return trees, nil
}

func AvailableSnapshotsHeights(dbpath string) (map[uint64]SnapshotData, error) {
	options := &opt.Options{
		ErrorIfMissing: true,
		ReadOnly:       true,
	}
	db, err := db.NewGoLevelDBWithOpts("snapshot", dbpath, options)
	if err != nil {
		return nil, fmt.Errorf("failed to open database located at %s : %w", dbpath, err)
	}

	tree, err := iavl.NewMutableTree(db, 0)
	if err != nil {
		return nil, err
	}

	if _, err := tree.Load(); err != nil {
		return nil, err
	}

	return SnapshotsHeightsFromTree(tree)
}
