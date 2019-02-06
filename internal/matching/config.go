package matching

import "vega/internal/logging"

type Config struct {
	log logging.Logger
	level logging.Level

	ProRataMode bool
	LogPriceLevelsDebug bool
	LogRemovedOrdersDebug bool
}

func NewConfig() *Config {
	level := logging.DebugLevel
	logger := logging.NewLogger()
	logger.InitConsoleLogger(level)
	logger.AddExitHandler()
	return &Config{
		log: logger,
		level: level,

		ProRataMode: false,
		LogPriceLevelsDebug: false,
		LogRemovedOrdersDebug: false,
	}
}

//func DefaultConfig() *Config {
//	conf := NewConfig()
//	conf.ProRataMode = false
//	return conf
//}

func ProRataModeConfig() *Config {
	conf := NewConfig()
	conf.ProRataMode = true
	return conf
}