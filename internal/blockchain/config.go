package blockchain

import (
	"vega/internal/logging"
)

type Config struct {
	log logging.Logger
	level logging.Level

	logTimeInfo bool
	logOrderSubmitDebug bool
	logOrderAmendDebug bool
	logOrderCancelDebug bool

	port int
	ip string
}

func NewConfig(logger logging.Logger) *Config {
	level := logging.DebugLevel
	logger = logger.Named("blockchain")
	return &Config{
		log: logger,
		level: level,
		port: 46658,
		ip: "localhost",

		logOrderSubmitDebug: true,
	}
}