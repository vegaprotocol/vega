package config

import (
	cfgencoding "code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
)

const (
	CandleStoreDataPath = "candlestore"
)

type CandlesConfig struct {
	Level   cfgencoding.LogLevel
	Storage StorageConfig
}

// NewDefaultCandlesConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultCandlesConfig(defaultConfigDirPath string) CandlesConfig {
	return CandlesConfig{
		Level:   cfgencoding.LogLevel{Level: logging.InfoLevel},
		Storage: newDefaultCandleStorageConfig(defaultConfigDirPath),
	}
}

func newDefaultCandleStorageConfig(defaultConfigDirPath string) StorageConfig {
	sc := newDefaultStorageConfig(defaultConfigDirPath, CandleStoreDataPath)

	// Add store-specific badger settings here.
	// sc.Badger.SomeKey = SomeValue
	// sc.Timeout = cfgencoding.Duration{Duration: N * time.Millisecond}

	return sc
}
