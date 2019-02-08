package api

import (
	"vega/internal/logging"
)

type Config struct {
	log logging.Logger
	level logging.Level

	GraphQLServerPort int
	GraphQLServerIpAddress string
	RestProxyServerPort int
	RestProxyIpAddress string
	GrpcServerPort int
	GrpcServerIpAddress string
}

func NewConfig(logger logging.Logger) *Config {
	level := logging.DebugLevel
	logger = logger.Named("api")
	return &Config{
		log: logger,
		level: level,

		GraphQLServerIpAddress: "127.0.0.1",
		GraphQLServerPort: 3004,

		RestProxyIpAddress: "127.0.0.1",
		RestProxyServerPort: 3003,

		GrpcServerIpAddress: "127.0.0.1",
		GrpcServerPort: 3002,
	}
}

func (c *Config) GetLogger() *logging.Logger {
	return &c.log
}