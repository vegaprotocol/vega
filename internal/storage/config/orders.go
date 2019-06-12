package config

import (
	cfgencoding "code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
)

const (
	OrderStoreDataPath = "orderstore"
)

type OrdersConfig struct {
	Level   cfgencoding.LogLevel
	Storage StorageConfig
}

// NewDefaultOrdersConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultOrdersConfig(defaultConfigDirPath string) OrdersConfig {
	return OrdersConfig{
		Level:   cfgencoding.LogLevel{Level: logging.InfoLevel},
		Storage: newDefaultOrderStorageConfig(defaultConfigDirPath),
	}
}

func newDefaultOrderStorageConfig(defaultConfigDirPath string) StorageConfig {
	sc := newDefaultStorageConfig(defaultConfigDirPath, OrderStoreDataPath)

	// Add store-specific badger settings here.
	// sc.Badger.SomeKey = SomeValue
	// sc.Timeout = cfgencoding.Duration{Duration: N * time.Millisecond}

	return sc
}
