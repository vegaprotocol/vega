package eth

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

type Config struct {
	Level       encoding.LogLevel `long:"log-level"`
	Address     string            `long:"address"`
	ClefAddress string            `long:"clef-address" description:"Clef address of running Clef instance. Clef wallet is used if defined"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
	}
}
