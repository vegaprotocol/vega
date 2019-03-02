package risk

import "vega/internal/logging"

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "risk"

type Config struct {
	log   *logging.Logger
	Level logging.Level

	// If set to true, all python risk model files will be loaded via an absolute path.
	// If set to false (default) all python risk model files will be loaded via relative path to the vega binary.
	PyRiskModelAbsolutePath    bool
	PyRiskModelDefaultFileName string
	PyRiskModelShortIndex      int
	PyRiskModelLongIndex       int
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig(logger *logging.Logger) *Config {
	logger = logger.Named(namedLogger)
	return &Config{
		log:                        logger,
		Level:                      logging.FatalLevel, //.InfoLevel,
		PyRiskModelDefaultFileName: "/risk-model.py",
		PyRiskModelShortIndex:      0,
		PyRiskModelLongIndex:       1,
		PyRiskModelAbsolutePath:    false,
	}
}

// GetLogger returns a pointer to the current underlying logger for this package.
func (c *Config) GetLogger() *logging.Logger {
	return c.log
}

// UpdateLogger will set any new values on the underlying logging core. Useful when configs are
// hot reloaded at run time. Currently we only check and refresh the logging level.
func (c *Config) UpdateLogger() {
	c.log.SetLevel(c.Level)
}
