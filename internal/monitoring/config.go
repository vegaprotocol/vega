package monitoring

import (
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
)

const (
	namedLogger = "monitoring"
)

type Config struct {
	log                  *logging.Logger
	Level                logging.Level
	IntervalMilliseconds time.Duration
	Retries              uint8
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig(log *logging.Logger) *Config {
	return &Config{
		log:                  log.Named(namedLogger),
		IntervalMilliseconds: 500, // this will be 500*time.Milliseconds when instanciated
		Retries:              5,
	}
}

// SetLogger creates a new logger based on a given parent logger.
func (c *Config) SetLogger(parent *logging.Logger) {
	c.log = parent.Named(namedLogger)
}

// UpdateLogger will set any new values on the underlying logging core. Useful when configs are
// hot reloaded at run time. Currently we only check and refresh the logging level.
func (c *Config) UpdateLogger() {
	c.log.SetLevel(c.Level)
}
