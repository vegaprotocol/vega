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
		// TODO Enable this validation error once the migration to 0.73 is done
		// 		on mainnet. In the meantime, we set it to 0 as this will save the
		//		validators to update their configuration from -1 to 0, which would
		//		trigger the removal of the snapshots in case of a rollback.
		//		In previous version, setting it to 0 is interpreted as starting
		//		from scratch.
		// return ErrStartHeightCannotBeNegative
		c.StartHeight = 0
	}

	switch c.Storage {
	case InMemoryDB, LevelDB:
		return nil
	default:
		return types.ErrInvalidSnapshotStorageMethod
	}
}
