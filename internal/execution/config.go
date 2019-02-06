package execution

import "vega/internal/logging"

type Config struct {
	log logging.Logger
}

func NewConfig() *Config {
	level := logging.DebugLevel
	logger := logging.NewLogger()
	logger.InitConsoleLogger(level)
	logger.AddExitHandler()
	return &Config{
		log: logger,
	}
}

