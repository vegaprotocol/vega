package monitoring

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "monitoring"
)

// Config represents the configuration of the monitoring package
type Config struct {
	Level    encoding.LogLevel `long:"log-level" description:" "`
	Interval encoding.Duration `long:"interval" description:" "`
	Retries  uint8             `long:"retries" description:" "`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:    encoding.LogLevel{Level: logging.InfoLevel},
		Interval: encoding.Duration{Duration: 500 * time.Millisecond}, // this will be 500*time.Milliseconds when instantiated
		Retries:  5,
	}
}
