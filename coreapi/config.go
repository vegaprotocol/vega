package coreapi

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "coreapi"
)

type Config struct {
	LogLevel encoding.LogLevel
	Accounts bool
	Assets   bool
}

func NewDefaultConfig() Config {
	return Config{
		LogLevel: encoding.LogLevel{Level: logging.InfoLevel},
		Accounts: true,
		Assets:   true,
	}
}
