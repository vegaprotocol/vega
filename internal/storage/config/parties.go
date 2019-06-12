package config

import (
	cfgencoding "code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"

	"github.com/dgraph-io/badger/options"
)

const (
	PartyStoreDataPath = "partystore"
)

type PartiesConfig struct {
	Level   cfgencoding.LogLevel
	Storage StorageConfig
}

// NewDefaultPartiesConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultPartiesConfig(defaultConfigDirPath string) PartiesConfig {
	return PartiesConfig{
		Level:   cfgencoding.LogLevel{Level: logging.InfoLevel},
		Storage: newDefaultPartyStorageConfig(defaultConfigDirPath),
	}
}

func newDefaultPartyStorageConfig(defaultConfigDirPath string) StorageConfig {
	sc := newDefaultStorageConfig(defaultConfigDirPath, PartyStoreDataPath)

	// Add store-specific badger settings here.
	// sc.Badger.SomeKey = SomeValue
	// sc.Timeout = cfgencoding.Duration{Duration: N * time.Millisecond}

	// PartyStore, if/when it becomes a BadgerStore, will be in-memory.
	inmem := cfgencoding.FileLoadingMode{FileLoadingMode: options.MemoryMap}
	sc.Badger.TableLoadingMode = inmem
	sc.Badger.ValueLogLoadingMode = inmem

	return sc
}
