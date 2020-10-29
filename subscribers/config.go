package subscribers

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "subscribers"
)

// Config represent the configuration of the subscribers package
type Config struct {
	OrderEventLogLevel  encoding.LogLevel `long:"order-event-log-level"`
	MarketEventLogLevel encoding.LogLevel `long:"market-even-log-level"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		MarketEventLogLevel: encoding.LogLevel{Level: logging.InfoLevel},
		OrderEventLogLevel:  encoding.LogLevel{Level: logging.InfoLevel},
	}
}
