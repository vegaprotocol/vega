package banking

import (
	"time"

	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

const (
	namedLogger = "banking"
)

// Config represents governance specific configuration
type Config struct {
	// logging level
	Level            encoding.LogLevel `long:"log-level"`
	WithdrawalExpiry encoding.Duration `long:"withdrawal-expiry"`
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level:            encoding.LogLevel{Level: logging.InfoLevel},
		WithdrawalExpiry: encoding.Duration{Duration: 24 * time.Hour},
	}
}
