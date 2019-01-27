package storage

import "vega/internal/logging"

type Config struct {
	log                logging.Logger
	level              logging.Level
	orderStoreDirPath  string
	tradeStoreDirPath  string
	logPartyStoreDebug bool
	logOrderStoreDebug bool

}

func NewConfig() *Config {
	level := logging.DebugLevel
	logger := logging.NewLogger()
	logger.InitConsoleLogger(level)
	logger.AddExitHandler()
	return &Config{
		log:                logger,
		level:              level,
		orderStoreDirPath: "../tmp/orderstore",
		tradeStoreDirPath: "../tmp/tradestore",
		logPartyStoreDebug: true,
		logOrderStoreDebug: true,

	}
}
