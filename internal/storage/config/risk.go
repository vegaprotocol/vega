package config

import (
	cfgencoding "code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"

	"github.com/dgraph-io/badger/options"
)

const (
	RiskStoreDataPath = "riskstore"
)

type RiskConfig struct {
	Level           cfgencoding.LogLevel
	LogMarginUpdate bool
	Storage         StorageConfig
}

// NewDefaultRiskConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultRiskConfig(defaultConfigDirPath string) RiskConfig {
	return RiskConfig{
		Level:           cfgencoding.LogLevel{Level: logging.InfoLevel},
		LogMarginUpdate: true,
		Storage:         newDefaultRiskStorageConfig(defaultConfigDirPath),
	}
}

func newDefaultRiskStorageConfig(defaultConfigDirPath string) StorageConfig {
	sc := newDefaultStorageConfig(defaultConfigDirPath, RiskStoreDataPath)

	// Add store-specific badger settings here.
	// sc.Badger.SomeKey = SomeValue
	// sc.Timeout = cfgencoding.Duration{Duration: N * time.Millisecond}

	// RiskStore, if/when it becomes a BadgerStore, will be in-memory.
	inmem := cfgencoding.FileLoadingMode{FileLoadingMode: options.MemoryMap}
	sc.Badger.TableLoadingMode = inmem
	sc.Badger.ValueLogLoadingMode = inmem

	return sc
}
