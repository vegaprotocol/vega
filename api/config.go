package api

import (
	"vega/internal/logging"
)

type Config struct {
	log logging.Logger
	level logging.Level

	GraphQLServerPort int
	GraphQLServerIpAddress string
	GrpcServerPort int
	GrpcServerIpAddress string
	RestProxyServerPort int
	RestProxyIpAddress string
}

func NewConfig() *Config {
	level := logging.DebugLevel
	logger := logging.NewLogger()
	logger.InitConsoleLogger(level)
	logger.AddExitHandler()
	return &Config{
		log: logger,
		level: level,

		GraphQLServerIpAddress: "127.0.0.1",
		GraphQLServerPort: 3000,
		GrpcServerIpAddress: "127.0.0.1",
		GrpcServerPort: 3002,
		RestProxyIpAddress: "127.0.0.1",
		RestProxyServerPort: 3001,
	}
}

func (c *Config) GetLogger() *logging.Logger {
	return &c.log
}