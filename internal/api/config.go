package api

import (
	"code.vegaprotocol.io/vega/internal/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "api"

type Config struct {
	log   *logging.Logger
	Level logging.Level

	GraphQLServerPort      int
	GraphQLServerIpAddress string
	RestProxyServerPort    int
	RestProxyIpAddress     string
	GrpcServerPort         int
	GrpcServerIpAddress    string
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig(logger *logging.Logger) *Config {
	logger = logger.Named(namedLogger)
	return &Config{
		log:   logger,
		Level: logging.InfoLevel,

		GraphQLServerIpAddress: "0.0.0.0",
		GraphQLServerPort:      3004,

		RestProxyIpAddress:  "0.0.0.0",
		RestProxyServerPort: 3003,

		GrpcServerIpAddress: "0.0.0.0",
		GrpcServerPort:      3002,
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
