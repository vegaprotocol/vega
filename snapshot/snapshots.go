package snapshot

import (
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"github.com/cosmos/iavl"
	"github.com/syndtr/goleveldb/leveldb/opt"
	db "github.com/tendermint/tm-db"
)

// SnapshotData is a representation of the information we an scrape from the avl tree.
type SnapshotData struct {
	Version int64  `json:"version"`
	Hash    []byte `json:"hash"`
	Height  uint64 `json:"height"`
	Size    int64  `json:"size"`
}

func SnapshotsHeightsFromTree(tree *iavl.MutableTree) ([]SnapshotData, error) {
	trees := make([]SnapshotData, 0, 4)
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

		trees = append(trees, SnapshotData{
			Version: v,
			Height:  app.AppState.Height,
			Hash:    snap.Hash,
			Size:    tree.Size(),
		})
	}
	sort.SliceStable(trees, func(i, j int) bool {
		return trees[i].Height > trees[j].Height
	})

	return trees, nil
}

func AvailableSnapshotsHeights(dbpath string) ([]SnapshotData, error) {
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
