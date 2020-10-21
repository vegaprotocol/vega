package netparams

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "netparams"

type Config struct {
	// logging level
	Level encoding.LogLevel
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
	}
}
