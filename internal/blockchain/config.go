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

	ClientAddr          string `mapstructure:"client_addr"`
	ClientEndpoint      string `mapstructure:"server_port"`
	ServerPort          int    `mapstructure:"server_port"`
	ServerAddr          string `mapstructure:"server_addr"`
	LogTimeDebug        bool   `mapstructure:"time_debug"`
	LogOrderSubmitDebug bool   `mapstructure:"order_submit_debug"`
	LogOrderAmendDebug  bool   `mapstructure:"order_amend_debug"`
	LogOrderCancelDebug bool   `mapstructure:"order_cancel_debug"`
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
