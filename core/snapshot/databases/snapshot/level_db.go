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
