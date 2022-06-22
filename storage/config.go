// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package storage

import (
	"time"

	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

const (
	// namedLogger is the identifier for package and should ideally match the package name
	// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
	namedLogger = "storage"

	defaultStorageAccessTimeout = 5 * time.Second
)

// Config provides package level settings, configuration and logging.
type Config struct {
	Accounts    ConfigOptions
	Candles     ConfigOptions
	Checkpoints ConfigOptions
	Markets     ConfigOptions
	Orders      ConfigOptions
	Trades      ConfigOptions
	// Parties   ConfigOptions  // Further badger store or hybrid store options
	// Depth     ConfigOptions  // will go here in the future (examples shown)
	// Risk      ConfigOptions
	// Positions ConfigOptions

	Level encoding.LogLevel `long:"log-level"`

	Timeout encoding.Duration `long:"timeout"`

	LogPositionStoreDebug bool `long:"log-position-store-debug"`
}

// NewDefaultConfig constructs a new Config instance with default parameters.
// This constructor is used by the vega application code. Logger is a
// pointer to a logging instance and defaultStoreDirPath is the root directory
// where all storage directories are to be read from and written to.
func NewDefaultConfig() Config {
	return Config{
		Accounts:              DefaultStoreOptions(),
		Candles:               DefaultStoreOptions(),
		Checkpoints:           DefaultStoreOptions(),
		Markets:               DefaultMarketStoreOptions(),
		Orders:                DefaultStoreOptions(),
		Trades:                DefaultStoreOptions(),
		Level:                 encoding.LogLevel{Level: logging.WarnLevel},
		LogPositionStoreDebug: false,
		Timeout:               encoding.Duration{Duration: defaultStorageAccessTimeout},
	}
}

// NewTestConfig constructs a new Config instance with test parameters.
// This constructor is exclusively used in unit tests/integration tests
func NewTestConfig() (Config, error) {
	// Test configuration for badger stores
	cfg := Config{
		Accounts:              DefaultStoreOptions(),
		Candles:               DefaultStoreOptions(),
		Markets:               DefaultStoreOptions(),
		Orders:                DefaultStoreOptions(),
		Trades:                DefaultStoreOptions(),
		LogPositionStoreDebug: true,
		Timeout:               encoding.Duration{Duration: defaultStorageAccessTimeout},
	}

	return cfg, nil
}
