package service

import (
	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

// Config represent the configuration of the candle package
type Config struct {
	Level encoding.LogLevel `long:"log-level"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
	}
}
