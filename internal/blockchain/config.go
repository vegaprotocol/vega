package blockchain

import (
	"vega/internal/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "blockchain"

type Config struct {
	log   *logging.Logger
	Level logging.Level

	ClientAddr          string
	ClientEndpoint      string
	ServerPort          int
	ServerAddr          string
	LogTimeDebug        bool
	LogOrderSubmitDebug bool
	LogOrderAmendDebug  bool
	LogOrderCancelDebug bool
}

func NewConfig(logger *logging.Logger) *Config {
	logger = logger.Named(namedLogger)
	return &Config{
		log:                 logger,
		Level:               logging.InfoLevel,
		ServerPort:          46658,
		ServerAddr:          "localhost",
		ClientAddr:          "tcp/://0.0.0.0:46657",
		ClientEndpoint:      "/websocket",
		LogOrderSubmitDebug: true,
		LogTimeDebug:        true,
	}
}
