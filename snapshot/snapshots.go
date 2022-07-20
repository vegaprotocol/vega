// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package snapshot

import (
	"fmt"
	"sort"

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

func SnapshotsHeightsFromTree(tree *iavl.MutableTree) ([]SnapshotData, []SnapshotData, error) {
	trees := make([]SnapshotData, 0, 4)
	invalidVersions := make([]SnapshotData, 0, 4)
	versions := tree.AvailableVersions()

	for _, version := range versions {
		v, err := tree.LazyLoadVersion(int64(version))
		if err != nil {
			return nil, nil, err
		}

		app, err := types.AppStateFromTree(tree.ImmutableTree)
		if err != nil {
			invalidVersions = append(invalidVersions, SnapshotData{
				Version: v,
				Hash:    tree.Hash(),
			})
			continue
		}

		snap, err := types.SnapshotFromTree(tree.ImmutableTree)
		if err != nil {
			return nil, nil, err
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

	return trees, invalidVersions, nil
}

func AvailableSnapshotsHeights(dbpath string) ([]SnapshotData, []SnapshotData, error) {
	options := &opt.Options{
		ErrorIfMissing: true,
		ReadOnly:       true,
	}
	db, err := db.NewGoLevelDBWithOpts("snapshot", dbpath, options)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database located at %s : %w", dbpath, err)
	}

	tree, err := iavl.NewMutableTree(db, 0)
	if err != nil {
		return nil, nil, err
	}

	if _, err := tree.Load(); err != nil {
		return nil, nil, err
	}

	return SnapshotsHeightsFromTree(tree)
}
