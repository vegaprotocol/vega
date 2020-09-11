package banking

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "banking"
)

// Config represents governance specific configuration
type Config struct {
	// logging level
	Level            encoding.LogLevel
	WithdrawalExpiry encoding.Duration
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level:            encoding.LogLevel{Level: logging.InfoLevel},
		WithdrawalExpiry: encoding.Duration{Duration: 24 * time.Hour},
	}
}
