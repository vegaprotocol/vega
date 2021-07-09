package validators

import (
	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

const (
	namedLogger = "validators"
)

// Config represents governance specific configuration
type Config struct {
	// logging level
	Level encoding.LogLevel `long:"log-level"`
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
	}
}
