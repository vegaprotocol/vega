package processor

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "processor"

	minValidationPeriod = 600       // ten minutes
	maxValidationPeriod = 48 * 3600 // 2 days
	nodeApproval        = 1         // float for percentage
)

// Config represent the configuration of the processor package
type Config struct {
	Level               encoding.LogLevel
	LogOrderSubmitDebug bool
	LogOrderAmendDebug  bool
	LogOrderCancelDebug bool
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:               encoding.LogLevel{Level: logging.InfoLevel},
		LogOrderSubmitDebug: true,
	}
}
