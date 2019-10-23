package noop

import (
	"time"

	"code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
)

const namedLogger = "noop"

type Config struct {
	Level         encoding.LogLevel
	BlockDuration encoding.Duration
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:         encoding.LogLevel{Level: logging.InfoLevel},
		BlockDuration: encoding.Duration{Duration: 1 * time.Second},
	}
}
