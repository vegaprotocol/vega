package engines

import (
	"code.vegaprotocol.io/vega/internal/engines/matching"
	"code.vegaprotocol.io/vega/internal/logging"
)

const (
	// namedLogger is the identifier for package and should ideally match the package name
	// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
	namedLogger = "engines"
)

type Config struct {
	log   *logging.Logger
	Level logging.Level

	Matching *matching.Config
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig(logger *logging.Logger) *Config {
	logger = logger.Named(namedLogger)
	return &Config{
		log:      logger,
		Level:    logging.InfoLevel,
		Matching: matching.NewDefaultConfig(logger),
	}
}

// GetLogger returns a pointer to the current underlying logger for this package.
func (c *Config) GetLogger() *logging.Logger {
	return c.log
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
