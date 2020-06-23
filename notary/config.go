package notary

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "notary"

	defaultSignatureRequired = 1.0
)

// Config represents governance specific configuration
type Config struct {
	// logging level
	Level                     encoding.LogLevel
	SignaturesRequiredPercent float64
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level:                     encoding.LogLevel{Level: logging.InfoLevel},
		SignaturesRequiredPercent: defaultSignatureRequired,
	}
}
