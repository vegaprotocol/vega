package snapshot

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "snapshot"
	goLevelDB   = "GOLevelDB"
	memDB       = "memory"
)

type Config struct {
	Level      encoding.LogLevel `long:"log-level"`
	Versions   int               `long:"versions"`
	RetryLimit int               `long:"max-retries"`
	Storage    string            `long:"storage"`
	DBPath     string            `long:"db-path"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:      encoding.LogLevel{Level: logging.InfoLevel},
		Versions:   10,
		RetryLimit: 5,
		Storage:    goLevelDB,
	}
}

func NewTestConfig() Config {
	cfg := NewDefaultConfig()
	cfg.Storage = memDB
	return cfg
}
