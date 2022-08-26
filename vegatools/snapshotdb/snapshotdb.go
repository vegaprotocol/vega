package snapshotdb

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/cosmos/iavl"
	"github.com/gogo/protobuf/jsonpb"
	pb "github.com/gogo/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb/opt"
	db "github.com/tendermint/tm-db"
	"google.golang.org/protobuf/proto"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/paths"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

type Showing byte

const (
	ShowList = iota
	ShowJSON
	ShowVersions
	UndefinedShowing
)

func ShowingFromString(s string) Showing {
	s = strings.ToLower(s)
	switch s {
	case "json":
		return ShowJSON
	case "list", "":
		return ShowList
	case "versions":
		return ShowVersions
	default:
		log.Fatalf("invalid show option: %s", s)
		return UndefinedShowing
	}
}

func ShowSnapshotData(dbPath, vegaHome string, show Showing, outputPath string, heightToOutput uint64) error {
	if dbPath == "" {
		vegaPaths := paths.New(vegaHome)
		dbPath = vegaPaths.StatePathFor(paths.SnapshotStateHome)
	} else {
		dbPath = paths.StatePath(dbPath).String()
	}

	options := &opt.Options{
		ErrorIfMissing: true,
		ReadOnly:       true,
	}

	dbc, err := db.NewGoLevelDBWithOpts("snapshot", dbPath, options)
	if err != nil {
		return fmt.Errorf("failed to open database located at %s : %w", dbPath, err)
	}

	tree, err := iavl.NewMutableTree(dbc, 0)
	if err != nil {
		return err
	}

	if _, err = tree.Load(); err != nil {
		return err
	}

	if outputPath != "" {
		log.Printf("Saving payloads to '%s' file...\n", outputPath)
		return SavePayloadsToFile(tree, heightToOutput, outputPath)
	}

	if err = writeToStd(tree, heightToOutput, show); err != nil {
		return err
	}

	return nil
}

func writeToStd(tree *iavl.MutableTree, heightToOutput uint64, show Showing) error {
	found, invalidVersions, err := SnapshotsHDataFromTree(tree, heightToOutput)
	if err != nil {
		return err
	}

	printData := printFuncs[show]

	if len(found) > 0 {
		log.Println("Snapshots available:", len(found))
		printData(found)
	} else {
		log.Println("No snapshots available")
	}

	if len(invalidVersions) > 0 {
		log.Println("Invalid versions:", len(invalidVersions))
		printData(invalidVersions)
	}
	return nil
}

func SnapshotsHDataFromTree(tree *iavl.MutableTree, heightToOutput uint64) ([]SnapshotData, []SnapshotData, error) {
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
			hash, _ := tree.Hash()
			invalidVersions = append(invalidVersions, SnapshotData{
				Version: v,
				Hash:    fmt.Sprintf("%x", hash),
			})
			continue
		}

		snap, err := types.SnapshotFromTree(tree.ImmutableTree)
		if err != nil {
			return nil, nil, err
		}

		data := SnapshotData{
			Version: v,
			Height:  &app.AppState.Height,
			Hash:    fmt.Sprintf("%x", snap.Hash),
			Size:    tree.Size(),
		}

		if heightToOutput > 0 {
			if app.AppState.Height == heightToOutput {
				return []SnapshotData{data}, nil, nil
			}
			continue
		}

		trees = append(trees, data)
	}
	sort.SliceStable(trees, func(i, j int) bool {
		return *trees[i].Height > *trees[j].Height
	})

	return trees, invalidVersions, nil
}

// SnapshotData is a representation of the information we scrape from the avl tree.
type SnapshotData struct {
	Height  *uint64 `json:"height,omitempty"`
	Version int64   `json:"version"`
	Size    int64   `json:"size"`
	Hash    string  `json:"hash"`
}

var printFuncs = map[Showing]func([]SnapshotData){
	ShowList:     printSnapshotDataAsList,
	ShowVersions: printSnapshotVersions,
	ShowJSON:     printSnapshotDataAsJSON,
}

func printSnapshotDataAsList(snapshots []SnapshotData) {
	for _, snap := range snapshots {
		if snap.Height != nil {
			log.Printf("\tHeight %d, ", *snap.Height)
		} else {
			log.Print("\t")
		}
		log.Printf("Version: %d, Size %d, Hash: %s\n", snap.Version, snap.Size, snap.Hash)
	}
}

func printSnapshotVersions(snapshots []SnapshotData) {
	log.Printf("Block Heights: ")

	for i, snap := range snapshots {
		if snap.Height == nil {
			continue
		}
		log.Printf("%d", *snap.Height)
		if i < len(snapshots)-1 {
			log.Printf(", ")
		}
	}
	log.Println()
}

func printSnapshotDataAsJSON(snapshots []SnapshotData) {
	b, _ := json.MarshalIndent(&snapshots, "", "	")
	log.Println(string(b))
}

func SavePayloadsToFile(tree *iavl.MutableTree, heightToOutput uint64, outputPath string) error {
	// traverse the tree and get the payloads
	payloads, err := getAllPayloads(tree, heightToOutput)
	if err != nil {
		return err
	}

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

func getAllPayloads(tree *iavl.MutableTree, heightToOutput uint64) (payloads []*snapshot.Payload, err error) {
	_, err = tree.Iterate(func(key []byte, val []byte) bool {
		p := new(snapshot.Payload)
		if err = proto.Unmarshal(val, p); err != nil {
			return true
		}

		if heightToOutput > 0 {
			if appState := p.GetAppState(); appState != nil && appState.GetHeight() == heightToOutput {
				payloads = []*snapshot.Payload{}
				return true
			}
			return false
		}

		payloads = append(payloads, p)
		return false
	})
	if err != nil {
		return
	}

	return payloads, err
}
