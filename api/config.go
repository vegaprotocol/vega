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

func (c *Config) GetLogger() *logging.Logger {
	return c.log
}
