package api

import (
	"vega/internal/logging"
)

type Config struct {
	log *logging.Logger
	Level logging.Level

	GraphQLServerPort int
	GraphQLServerIpAddress string
	RestProxyServerPort int
	RestProxyIpAddress string
	GrpcServerPort int
	GrpcServerIpAddress string
}

func NewConfig(logger *logging.Logger) *Config {
	logger = logger.Named("api")
	return &Config{
		log: logger,
		Level: logging.InfoLevel,

		GraphQLServerIpAddress: "127.0.0.1",
		GraphQLServerPort: 3004,

		RestProxyIpAddress: "127.0.0.1",
		RestProxyServerPort: 3003,

		GrpcServerIpAddress: "127.0.0.1",
		GrpcServerPort: 3002,
	}
}

func (c *Config) GetLogger() *logging.Logger {
	return c.log
}