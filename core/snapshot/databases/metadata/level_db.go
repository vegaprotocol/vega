package metadata

import (
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/paths"
	"github.com/syndtr/goleveldb/leveldb/opt"
	db "github.com/tendermint/tm-db"
)

const metaDBName = "snapshot_meta"

type LevelDBAdapter struct {
	filePath string

	underlyingAdapter *db.GoLevelDB
}

func (a *LevelDBAdapter) Save(version []byte, state []byte) error {
	return a.underlyingAdapter.Set(version, state)
}

func (a *LevelDBAdapter) Load(version []byte) ([]byte, error) {
	return a.underlyingAdapter.Get(version)
}

func (a *LevelDBAdapter) Close() error {
	return a.underlyingAdapter.Close()
}

func (a *LevelDBAdapter) Clear() error {
	if err := a.underlyingAdapter.Close(); err != nil {
		return fmt.Errorf("could not close the connection: %w", err)
	}

	if err := os.RemoveAll(a.filePath); err != nil {
		return fmt.Errorf("could not remove the database file: %w", err)
	}

	underlyingAdapter, err := initializeUnderlyingAdapter(a.filePath)
	if err != nil {
		return err
	}
	a.underlyingAdapter = underlyingAdapter

	return nil
}

func NewLevelDBAdapter(vegaPaths paths.Paths) (*LevelDBAdapter, error) {
	filePath := vegaPaths.StatePathFor(paths.SnapshotMetadataDBStateFile)

	underlyingAdapter, err := initializeUnderlyingAdapter(filePath)
	if err != nil {
		return nil, err
	}

	return &LevelDBAdapter{
		filePath:          filePath,
		underlyingAdapter: underlyingAdapter,
	}, nil
}

func initializeUnderlyingAdapter(filePath string) (*db.GoLevelDB, error) {
	underlyingAdapter, err := db.NewGoLevelDBWithOpts(
		metaDBName,
		filePath,
		&opt.Options{
			BlockCacher:     opt.NoCacher,
			OpenFilesCacher: opt.NoCacher,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize LevelDB adapter: %w", err)
	}
	return underlyingAdapter, nil
}
