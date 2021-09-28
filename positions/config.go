package positions

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "position"

// Config represents the configuration of the position engine
type Config struct {
	Level             encoding.LogLevel `long:"log-level"`
	LogPositionUpdate encoding.Bool     `long:"log-position-update"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:             encoding.LogLevel{Level: logging.InfoLevel},
		LogPositionUpdate: true,
	}
}
