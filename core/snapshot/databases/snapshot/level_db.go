package snapshot

import (
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/paths"
	cometbftdb "github.com/cometbft/cometbft-db"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const dbName = "snapshot"

type LevelDBDatabase struct {
	*cometbftdb.GoLevelDB

	dbFile      string
	dbDirectory string
}

func (d *LevelDBDatabase) Clear() error {
	if err := d.GoLevelDB.Close(); err != nil {
		return fmt.Errorf("could not close the connection: %w", err)
	}

	if err := os.RemoveAll(d.dbFile); err != nil {
		return fmt.Errorf("could not remove the database file: %w", err)
	}

	adapter, err := initializeUnderlyingAdapter(d.dbDirectory)
	if err != nil {
		return err
	}
	d.GoLevelDB = adapter

	return nil
}

func NewLevelDBDatabase(vegaPaths paths.Paths) (*LevelDBDatabase, error) {
	dbDirectory := vegaPaths.StatePathFor(paths.SnapshotStateHome)

	// This has to be in sync with the `dbName` constant.
	dbFile := vegaPaths.StatePathFor(paths.SnapshotDBStateFile)

	adapter, err := initializeUnderlyingAdapter(dbDirectory)
	if err != nil {
		return nil, err
	}

	return &LevelDBDatabase{
		dbFile:      dbFile,
		dbDirectory: dbDirectory,
		GoLevelDB:   adapter,
	}, nil
}

func initializeUnderlyingAdapter(dbDirectory string) (*cometbftdb.GoLevelDB, error) {
	adapter, err := cometbftdb.NewGoLevelDBWithOpts(
		dbName,
		dbDirectory,
		&opt.Options{
			Filter:          filter.NewBloomFilter(10),
			BlockCacher:     opt.NoCacher,
			OpenFilesCacher: opt.NoCacher,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize LevelDB adapter: %w", err)
	}

	return adapter, nil
}
