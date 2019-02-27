package api

import (
	"vega/internal/logging"
)

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

func NewConfig(logger *logging.Logger) *Config {
	logger = logger.Named("api")
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
	c.log.SetLevel(c.Level, true)
}
