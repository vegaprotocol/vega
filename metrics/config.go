package metrics

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// Config represents the configuration of the metric package.
type Config struct {
	Level   encoding.LogLevel `long:"log-level" description:" "`
	Timeout encoding.Duration `long:"timeout" description:" "`
	Port    int               `long:"port" description:" "`
	Path    string            `long:"path" description:" "`
	Enabled encoding.Bool     `long:"enabled" description:" "`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:   encoding.LogLevel{Level: logging.InfoLevel},
		Timeout: encoding.Duration{Duration: 5000 * time.Millisecond},

		Port:    2112,
		Path:    "/metrics",
		Enabled: false,
	}
}
