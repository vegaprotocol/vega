package matching

import "vega/internal/logging"

type Config struct {
	log logging.Logger
	level logging.Level

	ProRataMode bool
	LogPriceLevelsDebug bool
	LogRemovedOrdersDebug bool
}

func NewConfig(logger logging.Logger) *Config {
	level := logging.DebugLevel
	logger = logger.Named("matching")
	
	return &Config{
		log: logger,
		level: level,

		ProRataMode: false,
		LogPriceLevelsDebug: false,
		LogRemovedOrdersDebug: false,
	}
}

func ProRataModeConfig(logger logging.Logger) *Config {
	conf := NewConfig(logger)
	conf.ProRataMode = true
	return conf
}