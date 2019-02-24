package matching

import "vega/internal/logging"

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "matching"

type Config struct {
	log   *logging.Logger
	Level logging.Level

	ProRataMode           bool
	LogPriceLevelsDebug   bool
	LogRemovedOrdersDebug bool
}

func NewConfig(logger *logging.Logger) *Config {
	logger = logger.Named(namedLogger)
	return &Config{
		log:                   logger,
		Level:                 logging.InfoLevel,
		ProRataMode:           false,
		LogPriceLevelsDebug:   false,
		LogRemovedOrdersDebug: false,
	}
}

func ProRataModeConfig(logger *logging.Logger) *Config {
	conf := NewConfig(logger)
	conf.ProRataMode = true
	return conf
}
