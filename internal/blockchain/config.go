package blockchain

import (
	"vega/internal/logging"
)

type Config struct {
	log   logging.Logger
	level logging.Level

	logTimeInfo         bool
	logOrderSubmitDebug bool
	logOrderAmendDebug  bool
	logOrderCancelDebug bool

	ClientAddr     string
	ClientEndpoint string

	ServerPort int
	ServerAddr string
}

func NewConfig(logger logging.Logger) *Config {

	level := logging.DebugLevel
	logger = logger.Named("blockchain")
	return &Config{
		log:                 logger,
		level:               level,
		ServerPort:          46658,
		ServerAddr:          "localhost",
		ClientAddr:          "tcp://0.0.0.0:46657",
		ClientEndpoint:      "/websocket",
		logOrderSubmitDebug: true,
		logTimeInfo:         true,
	}
}
