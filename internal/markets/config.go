package markets

import "vega/internal/logging"

type Config struct {
	log logging.Logger
	level logging.Level
}

func NewConfig(logger logging.Logger) *Config {
	level := logging.DebugLevel
	logger = logger.Named("markets")
	return &Config{
		log: logger,
		level: level,
	}
}