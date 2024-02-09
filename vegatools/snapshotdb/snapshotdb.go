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

package snapshotdb

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"

	metadatadb "code.vegaprotocol.io/vega/core/snapshot/databases/metadata"
	snapshotdb "code.vegaprotocol.io/vega/core/snapshot/databases/snapshot"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/paths"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	cometbftdb "github.com/cometbft/cometbft-db"
	"github.com/cosmos/iavl"
	"github.com/gogo/protobuf/jsonpb"
	pb "github.com/gogo/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"google.golang.org/protobuf/proto"
)

// Data is a representation of the information we scrape from the avl tree.
type Data struct {
	Height  uint64 `json:"height,omitempty"`
	Version int64  `json:"version"`
	Size    int64  `json:"size"`
	Hash    string `json:"hash"`
}

type Database interface {
	cometbftdb.DB
	Clear() error
}

func initialiseTree(dbPath string) (*cometbftdb.GoLevelDB, *iavl.MutableTree, error) {
	conn, err := cometbftdb.NewGoLevelDBWithOpts(
		"snapshot",
		dbPath,
		&opt.Options{
			ErrorIfMissing: true,
		})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database located at %s : %w", dbPath, err)
	}

	tree, err := iavl.NewMutableTree(conn, 0, false)
	if err != nil {
		return nil, nil, err
	}

	if _, err = tree.Load(); err != nil {
		return nil, nil, err
	}
	return conn, tree, nil
}

// SnapshotData returns an overview of each snapshot saved to disk, namely the height and its hash.
func SnapshotData(dbPath string, heightToOutput uint64) ([]Data, []Data, error) {
	conn, tree, err := initialiseTree(dbPath)
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	if err != nil {
		return nil, nil, err
	}
	versions := tree.AvailableVersions()
	trees := make([]Data, 0, len(versions))
	invalidVersions := make([]Data, 0)

	for _, version := range versions {
		v, err := tree.LazyLoadVersion(int64(version))
		if err != nil {
			return nil, nil, err
		}

		snapshotHash, _ := tree.Hash()
		app, err := types.AppStateFromTree(tree.ImmutableTree)
		if err != nil {
			invalidVersions = append(invalidVersions, Data{
				Version: v,
				Hash:    hex.EncodeToString(snapshotHash),
			})
			continue
		}

		data := Data{
			Version: v,
			Height:  app.AppState.Height,
			Hash:    hex.EncodeToString(snapshotHash),
			Size:    tree.Size(),
		}

		if heightToOutput > 0 {
			if app.AppState.Height == heightToOutput {
				return []Data{data}, nil, nil
			}
		}
		trees = append(trees, data)
	}
	sort.SliceStable(trees, func(i, j int) bool {
		return trees[i].Height > trees[j].Height
	})

	return trees, invalidVersions, nil
}

// SavePayloadsToFile given a block height and file path writes all the payloads for that snapshot height
// to the file in json format.
func SavePayloadsToFile(dbPath string, outputPath string, heightToOutput uint64) error {
	conn, tree, err := initialiseTree(dbPath)
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	if err != nil {
		return err
	}

	versions := tree.AvailableVersions()
	for i := len(versions) - 1; i > -1; i-- {
		_, err := tree.LazyLoadVersion(int64(versions[i]))
		if err != nil {
			return err
		}

		// looking up the appstate directly by its key first then unmarshalling all payloads
		// is quicker
		app, err := types.AppStateFromTree(tree.ImmutableTree)
		if err != nil {
			return err
		}
		if app.AppState.Height != heightToOutput {
			continue
		}
		payloads, _, _ := getAllPayloads(tree)
		return writePayloads(payloads, outputPath)
	}

	return errors.New("failed to find snapshot for block-height")
}

func writePayloads(payloads []*snapshotpb.Payload, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	w := bufio.NewWriter(f)
	m := jsonpb.Marshaler{Indent: "    "}

	payloadData := struct {
		Data []*snapshotpb.Payload `json:"data,omitempty" protobuf:"bytes,1,rep,name=data"`
		pb.Message
	}{
		Data: payloads,
	}

	s, err := m.MarshalToString(&payloadData)
	if err != nil {
		return err
	}

	if _, err = w.WriteString(s); err != nil {
		return err
	}

	if err = w.Flush(); err != nil {
		return err
	}

	return nil
}

func getAllPayloads(tree *iavl.MutableTree) (payloads []*snapshotpb.Payload, blockHeight uint64, err error) {
	_, err = tree.Iterate(func(key []byte, val []byte) bool {
		p := new(snapshotpb.Payload)
		if err = proto.Unmarshal(val, p); err != nil {
			return true
		}

		if appState := p.GetAppState(); appState != nil {
			blockHeight = appState.GetHeight()
		}

		payloads = append(payloads, p)
		return false
	})
	if err != nil {
		return
	}

	return payloads, blockHeight, err
}

// SetProtocolUpgrade will take the latest snapshot in the tree and rewrites it with the protocolUpgrade flag set to
// true so that we can pretend that the snapshot was taken for a upgrade to help with testing.
func SetProtocolUpgrade(vegaPaths paths.Paths) error {
	snap, _ := snapshotdb.NewLevelDBDatabase(vegaPaths)
	meta, _ := metadatadb.NewLevelDBDatabase(vegaPaths)
	defer snap.Close()
	defer meta.Close()

	tree, err := iavl.NewMutableTree(snap, 0, false)
	if err != nil {
		return err
	}

	if _, err = tree.Load(); err != nil {
		return err
	}

	tree.LazyLoadVersion(-1)
	oldVersion := tree.Version()

	var perr error
	var k, v []byte

	_, err = tree.Iterate(func(key []byte, val []byte) bool {
		p := &snapshotpb.Payload{}
		if perr = proto.Unmarshal(val, p); perr != nil {
			return true
		}

		appState := p.GetAppState()
		if appState == nil {
			return false
		}

		// change the value
		appState.ProtocolUpgrade = true

		// marshal it up again
		pp := &snapshotpb.Payload{
			Data: &snapshotpb.Payload_AppState{
				AppState: appState,
			},
		}
		k = key
		v, perr = proto.Marshal(pp)
		return true
	})

	if err != nil {
		return fmt.Errorf("failed to traverse snapshot payloads: %w", err)
	}

	if perr != nil {
		return fmt.Errorf("failed to unpack appstate payload: %w", perr)
	}

	if _, err = tree.Set(k, v); err != nil {
		return fmt.Errorf("failed to save new appstate to snapshot: %w", err)
	}

	_, newVersion, err := tree.SaveVersion()
	if err != nil {
		return err
	}

	// delete the old version so that we do not have two with the same block-height
	tree.DeleteVersion(oldVersion)

	// update the version <-> block-height map in the meta-data
	metaSnap, err := meta.Load(oldVersion)
	if err != nil {
		return err
	}

	// delete the meta-data pegged against the old version
	if err := meta.Delete(oldVersion); err != nil {
		return err
	}

	// new it against the new version
	if err := meta.Save(newVersion, metaSnap); err != nil {
		return err
	}
	return nil
}
