package processor

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/processor/ratelimit"
)

const (
	namedLogger = "processor"
)

// Config represent the configuration of the processor package
type Config struct {
	Level               encoding.LogLevel `long:"log-level"`
	LogOrderSubmitDebug encoding.Bool     `long:"log-order-submit-debug"`
	LogOrderAmendDebug  encoding.Bool     `long:"log-order-amend-debug"`
	LogOrderCancelDebug encoding.Bool     `long:"log-order-cancel-debug"`
	Ratelimit           ratelimit.Config  `group:"Ratelimit" namespace:"ratelimit"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:               encoding.LogLevel{Level: logging.InfoLevel},
		LogOrderSubmitDebug: true,
		Ratelimit:           ratelimit.NewDefaultConfig(),
	}
}
