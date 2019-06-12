package config

import (
	cfgencoding "code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
)

const (
	TradeStoreDataPath = "tradestore"
)

type TradesConfig struct {
	Level                 cfgencoding.LogLevel
	LogPositionStoreDebug bool
	Storage               StorageConfig
}

// NewDefaultTradesConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultTradesConfig(defaultConfigDirPath string) TradesConfig {
	return TradesConfig{
		Level:                 cfgencoding.LogLevel{Level: logging.InfoLevel},
		LogPositionStoreDebug: false,
		Storage:               newDefaultTradeStorageConfig(defaultConfigDirPath),
	}
}

func newDefaultTradeStorageConfig(defaultConfigDirPath string) StorageConfig {
	sc := newDefaultStorageConfig(defaultConfigDirPath, TradeStoreDataPath)

	// Add store-specific badger settings here.
	// sc.Badger.SomeKey = SomeValue
	// sc.Timeout = cfgencoding.Duration{Duration: N * time.Millisecond}

	return sc
}
