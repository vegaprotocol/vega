package config

import (
	cfgencoding "code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
)

const (
	AccountStoreDataPath = "accountstore"
)

type AccountsConfig struct {
	Level   cfgencoding.LogLevel
	Storage StorageConfig
}

// NewDefaultAccountsConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultAccountsConfig(defaultConfigDirPath string) AccountsConfig {
	return AccountsConfig{
		Level:   cfgencoding.LogLevel{Level: logging.InfoLevel},
		Storage: newDefaultAccountsStorageConfig(defaultConfigDirPath),
	}
}

func newDefaultAccountsStorageConfig(defaultConfigDirPath string) StorageConfig {
	sc := newDefaultStorageConfig(defaultConfigDirPath, AccountStoreDataPath)

	// Add store-specific badger settings here.
	// sc.Badger.SomeKey = SomeValue
	// sc.Timeout = cfgencoding.Duration{Duration: N * time.Millisecond}

	return sc
}
