package metadata

import (
	"fmt"
	"os"
	"strconv"

	"code.vegaprotocol.io/vega/paths"
	cometbftdb "github.com/cometbft/cometbft-db"
	"github.com/syndtr/goleveldb/leveldb/opt"
	tmtypes "github.com/tendermint/tendermint/abci/types"

	"code.vegaprotocol.io/vega/libs/proto"
)

const metaDBName = "snapshot_meta"

type LevelDBDatabase struct {
	dbFile      string
	dbDirectory string

	underlyingAdapter *cometbftdb.GoLevelDB
}

func (d *LevelDBDatabase) Save(version int64, state *tmtypes.Snapshot) error {
	serializedVersion := strconv.FormatInt(version, 10)

	serializedState, err := proto.Marshal(state)
	if err != nil {
		return fmt.Errorf("could not serialize snaspshot state: %w", err)
	}

	return d.underlyingAdapter.Set([]byte(serializedVersion), serializedState)
}

func (d *LevelDBDatabase) Load(version int64) (*tmtypes.Snapshot, error) {
	serializedVersion := strconv.FormatInt(version, 10)

	serializedState, err := d.underlyingAdapter.Get([]byte(serializedVersion))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve metadata for key %q: %w", serializedVersion, err)
	} else if serializedState == nil && err == nil {
		return nil, noMetadataForSnapshotVersion(version)
	}

	snapshot := &tmtypes.Snapshot{}
	if err := proto.Unmarshal(serializedState, snapshot); err != nil {
		return nil, fmt.Errorf("could not deserialize snapshot state: %w", err)
	}

	return snapshot, err
}

func (d *LevelDBDatabase) Close() error {
	return d.underlyingAdapter.Close()
}

func (d *LevelDBDatabase) IsEmpty() bool {
	iter := d.underlyingAdapter.DB().NewIterator(nil, nil)
	defer iter.Release()
	return !iter.Next()
}

func (d *LevelDBDatabase) FindVersionByBlockHeight(blockHeight uint64) (int64, error) {
	iter := d.underlyingAdapter.DB().NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		snapshot := &tmtypes.Snapshot{}
		if err := proto.Unmarshal(iter.Value(), snapshot); err != nil {
			return -1, fmt.Errorf("could not deserialize snapshot state: %w", err)
		}

		if snapshot.Height == blockHeight {
			version, err := strconv.ParseInt(string(iter.Key()), 10, 64)
			if err != nil {
				return -1, fmt.Errorf("could not deserialize the snapshot version for block height %d: %w", blockHeight, err)
			}
			return version, nil
		}
	}
	if err := iter.Error(); err != nil {
		return -1, fmt.Errorf("an error occurred while iterating over the metadata: %w", err)
	}

	return -1, nil
}

func (d *LevelDBDatabase) Delete(version int64) error {
	serializedVersion := strconv.FormatInt(version, 10)

	if err := d.underlyingAdapter.Delete([]byte(serializedVersion)); err != nil {
		return fmt.Errorf("could not delete metadata for key %q: %w", serializedVersion, err)
	}

	return nil
}

func (d *LevelDBDatabase) DeleteRange(fromVersion, toVersion int64) error {
	iter := d.underlyingAdapter.DB().NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		version, err := strconv.ParseInt(string(iter.Key()), 10, 64)
		if err != nil {
			return fmt.Errorf("could not deserialize the version %q: %w", iter.Key(), err)
		}

		if version >= fromVersion && version < toVersion {
			if err := d.underlyingAdapter.Delete(iter.Key()); err != nil {
				return fmt.Errorf("could not delete metadata for key %q: %w", iter.Key(), err)
			}
		}
	}

	if err := iter.Error(); err != nil {
		return fmt.Errorf("an error occurred while iterating over the metadata: %w", err)
	}

	return nil
}

func (d *LevelDBDatabase) Clear() error {
	if err := d.underlyingAdapter.Close(); err != nil {
		return fmt.Errorf("could not close the connection: %w", err)
	}

	if err := os.RemoveAll(d.dbFile); err != nil {
		return fmt.Errorf("could not remove the database file: %w", err)
	}

	underlyingAdapter, err := initializeUnderlyingAdapter(d.dbDirectory)
	if err != nil {
		return err
	}
	d.underlyingAdapter = underlyingAdapter

	return nil
}

func NewLevelDBDatabase(vegaPaths paths.Paths) (*LevelDBDatabase, error) {
	dbDirectory := vegaPaths.StatePathFor(paths.SnapshotStateHome)

	// This has to be in sync with the `metaDBName` constant.
	dbFile := vegaPaths.StatePathFor(paths.SnapshotMetadataDBStateFile)

	underlyingAdapter, err := initializeUnderlyingAdapter(dbDirectory)
	if err != nil {
		return nil, err
	}

	return &LevelDBDatabase{
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
