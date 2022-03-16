package snapshot

import (
	"errors"
	"os"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
)

const (
	namedLogger = "snapshot"
	goLevelDB   = "GOLevelDB"
	memDB       = "memory"
)

type Config struct {
	Level       encoding.LogLevel `long:"log-level" choice:"debug" choice:"info" choice:"warning" choice:"error" choice:"panic" choice:"fatal" description:"Logging level (default: info)"`
	KeepRecent  int               `long:"snapshot-keep-recent" description:"Number of historic snapshots to keep on disk. Limited to the 10 most recent ones"`
	RetryLimit  int               `long:"max-retries" description:"Maximum number of times to try and apply snapshot chunk"`
	Storage     string            `long:"storage" choice:"GOLevelDB" choice:"memory" description:"Storage type to use"`
	DBPath      string            `long:"db-path" description:"Path to database"`
	StartHeight int64             `long:"load-from-block-height" description:"Start the node by loading the snapshot taken at the given block-height. -1 for last snapshot, 0 for no reload (default: 0)"` // -1 for last snapshot, 0 for no reload
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:       encoding.LogLevel{Level: logging.InfoLevel},
		KeepRecent:  10,
		RetryLimit:  5,
		Storage:     goLevelDB,
		StartHeight: 0,
	}
}

func NewTestConfig() Config {
	cfg := NewDefaultConfig()
	cfg.Storage = memDB
	return cfg
}

// validate checks the values in the config file are sensible, and returns the path
// which is create/load the snapshots from.
func (c *Config) validate(vegapath paths.Paths) (string, error) {
	if len(c.DBPath) != 0 && c.Storage == memDB {
		return "", errors.New("dbpath cannot be set when storage method is in-memory")
	}

	switch c.Storage {
	case memDB:
		return "", nil
	case goLevelDB:

		if len(c.DBPath) == 0 {
			return vegapath.StatePathFor(paths.SnapshotStateHome), nil
		}

		stat, err := os.Stat(c.DBPath)
		if err != nil {
			return "", err
		}

		if !stat.IsDir() {
			return "", errors.New("snapshot DB path is not a directory")
		}

		return c.DBPath, nil
	default:
		return "", types.ErrInvalidSnapshotStorageMethod
	}
}
