package snapshotdb

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/cosmos/iavl"
	"github.com/gogo/protobuf/jsonpb"
	pb "github.com/gogo/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb/opt"
	db "github.com/tendermint/tm-db"
	"google.golang.org/protobuf/proto"

	"code.vegaprotocol.io/vega/core/types"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

// Data is a representation of the information we scrape from the avl tree.
type Data struct {
	Height  uint64 `json:"height,omitempty"`
	Version int64  `json:"version"`
	Size    int64  `json:"size"`
	Hash    string `json:"hash"`
}

func initialiseTree(dbPath string) (*db.GoLevelDB, *iavl.MutableTree, error) {
	conn, err := db.NewGoLevelDBWithOpts(
		"snapshot",
		dbPath,
		&opt.Options{
			ErrorIfMissing: true,
			ReadOnly:       true,
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
func SnapshotData(dbPath string, outputPath string, heightToOutput uint64) ([]Data, []Data, error) {
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

func writePayloads(payloads []*snapshot.Payload, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	w := bufio.NewWriter(f)
	m := jsonpb.Marshaler{Indent: "    "}

	payloadData := struct {
		Data []*snapshot.Payload `protobuf:"bytes,1,rep,name=data" json:"data,omitempty"`
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

func getAllPayloads(tree *iavl.MutableTree) (payloads []*snapshot.Payload, blockHeight uint64, err error) {
	_, err = tree.Iterate(func(key []byte, val []byte) bool {
		p := new(snapshot.Payload)
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
