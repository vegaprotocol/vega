package ethcall

import (
	"time"

	"code.vegaprotocol.io/vega/core/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	defaultPollEvery = 20 * time.Second
)

type Config struct {
	Level     encoding.LogLevel `long:"log-level"`
	PollEvery encoding.Duration
}

func NewDefaultConfig() Config {
	return Config{
		Level:     encoding.LogLevel{Level: logging.InfoLevel},
		PollEvery: encoding.Duration{Duration: defaultPollEvery},
	}
}
