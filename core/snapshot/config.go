// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package snapshot

import (
	"errors"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	LevelDB    = "GOLevelDB"
	InMemoryDB = "memory"
)

var (
	ErrStartHeightCannotBeNegative      = errors.New("the value for \"load-from-block-height\" must be positive, or zero")
	ErrKeepRecentMustBeHigherOrEqualTo1 = errors.New("the value for \"snapshot-keep-recent\" must higher or equal to 1")
)

type Config struct {
	Level       encoding.LogLevel `choice:"debug"                                                                                                                                                                                                                                                                                            choice:"info"                 choice:"warning"                  choice:"error" choice:"panic" choice:"fatal" description:"Logging level (default: info)" long:"log-level"`
	KeepRecent  uint              `description:"Number of historic snapshots to keep on disk. The minimum value is 1."                                                                                                                                                                                                                       long:"snapshot-keep-recent"`
	RetryLimit  uint              `description:"Maximum number of times to try and apply snapshot chunk coming from state-sync"                                                                                                                                                                                                              long:"max-retries"`
	Storage     string            `choice:"GOLevelDB"                                                                                                                                                                                                                                                                                        choice:"memory"               description:"Storage type to use" long:"storage"`
	StartHeight int64             `description:"If there are local snapshots, load the one matching the specified block height. If there is no local snapshot, and state-sync is enabled, the node waits for a snapshot to match the specified block height to be offered by the network peers. If set to 0, the latest snapshot is loaded." long:"load-from-block-height"`
}

// DefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func DefaultConfig() Config {
	return Config{
		Level:       encoding.LogLevel{Level: logging.InfoLevel},
		KeepRecent:  10,
		RetryLimit:  5,
		Storage:     LevelDB,
		StartHeight: 0,
	}
}

// Validate checks the values in the config file are sensible.
func (c *Config) Validate() error {
	if c.KeepRecent < 1 {
		return ErrKeepRecentMustBeHigherOrEqualTo1
	}

	if c.StartHeight < 0 {
		return ErrStartHeightCannotBeNegative
	}

	switch c.Storage {
	case InMemoryDB, LevelDB:
		return nil
	default:
		return types.ErrInvalidSnapshotStorageMethod
	}
}
