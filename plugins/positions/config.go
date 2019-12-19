package positions

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	cfgencoding "code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// Config represents the configuration of the Orders service
type Config struct {
	Level              cfgencoding.LogLevel
	StreamRetries      int
	Timeout            encoding.Duration
	SubscriptionBuffer int
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func DefaultConfig() Config {
	return Config{
		Level:              cfgencoding.LogLevel{Level: logging.InfoLevel},
		Timeout:            encoding.Duration{Duration: 5000 * time.Millisecond},
		StreamRetries:      3,
		SubscriptionBuffer: 10,
	}
}
