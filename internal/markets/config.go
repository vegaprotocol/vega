package markets

import "vega/internal/logging"

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "markets"

type Config struct {
	log   *logging.Logger
	Level logging.Level
}

func NewConfig(logger *logging.Logger) *Config {
	logger = logger.Named(namedLogger)
	return &Config{
		log:   logger,
		Level: logging.InfoLevel,
	}
}
