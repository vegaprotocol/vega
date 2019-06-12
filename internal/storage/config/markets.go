package config

import (
	cfgencoding "code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
)

const (
	MarketStoreDataPath = "marketstore"
)

type MarketsConfig struct {
	Level   cfgencoding.LogLevel
	Storage StorageConfig
}

// NewDefaultMarketsConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultMarketsConfig(defaultConfigDirPath string) MarketsConfig {
	return MarketsConfig{
		Level:   cfgencoding.LogLevel{Level: logging.InfoLevel},
		Storage: newDefaultMarketStorageConfig(defaultConfigDirPath),
	}
}

func newDefaultMarketStorageConfig(defaultConfigDirPath string) StorageConfig {
	sc := newDefaultStorageConfig(defaultConfigDirPath, MarketStoreDataPath)

	// Add store-specific badger settings here.
	// sc.Badger.SomeKey = SomeValue
	// sc.Timeout = cfgencoding.Duration{Duration: N * time.Millisecond}

	return sc
}
