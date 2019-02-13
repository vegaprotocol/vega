package vegatime

import "vega/internal/logging"

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'. Note for readability
// I've called this package name 'time'.
const namedLogger = "time"

type Config struct {
	log logging.Logger
	level logging.Level
}

func NewConfig(logger logging.Logger) *Config {
	logger = logger.Named(namedLogger)
	level := logging.DebugLevel
	return &Config{
		log: logger,
		level: level,
	}
}