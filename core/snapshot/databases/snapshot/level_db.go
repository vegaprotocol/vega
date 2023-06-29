package metadata

import (
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/paths"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	db "github.com/tendermint/tm-db"
)

const dbName = "snapshot"

type LevelDBDatabase struct {
	*db.GoLevelDB

	filePath string
}

func (d *LevelDBDatabase) Clear() error {
	if err := d.GoLevelDB.Close(); err != nil {
		return fmt.Errorf("could not close the connection: %w", err)
	}

	if err := os.RemoveAll(d.filePath); err != nil {
		return fmt.Errorf("could not remove the database file: %w", err)
	}

	adapter, err := initializeUnderlyingAdapter(d.filePath)
	if err != nil {
		return err
	}
	d.GoLevelDB = adapter

	return nil
}

func NewLevelDBDatabase(vegaPaths paths.Paths) (*LevelDBDatabase, error) {
	filePath := vegaPaths.StatePathFor(paths.SnapshotStateHome)

	adapter, err := initializeUnderlyingAdapter(filePath)
	if err != nil {
		return nil, err
	}

	return &LevelDBDatabase{
		filePath:  filePath,
		GoLevelDB: adapter,
	}, nil
}

func initializeUnderlyingAdapter(filePath string) (*db.GoLevelDB, error) {
	adapter, err := db.NewGoLevelDBWithOpts(
		dbName, filePath,
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
