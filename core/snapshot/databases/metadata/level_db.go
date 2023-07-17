package metadata

import (
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/paths"
	cometbftdb "github.com/cometbft/cometbft-db"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const metaDBName = "snapshot_meta"

type LevelDBAdapter struct {
	dbFile      string
	dbDirectory string

	underlyingAdapter *cometbftdb.GoLevelDB
}

func (a *LevelDBAdapter) Save(version []byte, state []byte) error {
	return a.underlyingAdapter.Set(version, state)
}

func (a *LevelDBAdapter) Load(version []byte) ([]byte, error) {
	loadedData, err := a.underlyingAdapter.Get(version)
	if loadedData == nil && err == nil {
		return nil, noMetadataForSnapshotVersion(version)
	}
	return loadedData, err
}

func (a *LevelDBAdapter) Close() error {
	return a.underlyingAdapter.Close()
}

func (a *LevelDBAdapter) ContainsMetadata() bool {
	iter := a.underlyingAdapter.DB().NewIterator(nil, nil)
	defer iter.Release()
	return iter.Next()
}

func (a *LevelDBAdapter) Clear() error {
	if err := a.underlyingAdapter.Close(); err != nil {
		return fmt.Errorf("could not close the connection: %w", err)
	}

	if err := os.RemoveAll(a.dbFile); err != nil {
		return fmt.Errorf("could not remove the database file: %w", err)
	}

	underlyingAdapter, err := initializeUnderlyingAdapter(a.dbDirectory)
	if err != nil {
		return err
	}
	a.underlyingAdapter = underlyingAdapter

	return nil
}

func NewLevelDBAdapter(vegaPaths paths.Paths) (*LevelDBAdapter, error) {
	dbDirectory := vegaPaths.StatePathFor(paths.SnapshotStateHome)

	// This has to be in sync with the `metaDBName` constant.
	dbFile := vegaPaths.StatePathFor(paths.SnapshotMetadataDBStateFile)

	underlyingAdapter, err := initializeUnderlyingAdapter(dbDirectory)
	if err != nil {
		return nil, err
	}

	return &LevelDBAdapter{
		dbFile:            dbFile,
		dbDirectory:       dbDirectory,
		underlyingAdapter: underlyingAdapter,
	}, nil
}

func initializeUnderlyingAdapter(dbDirectory string) (*cometbftdb.GoLevelDB, error) {
	underlyingAdapter, err := cometbftdb.NewGoLevelDBWithOpts(
		metaDBName,
		dbDirectory,
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
