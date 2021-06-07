package eth

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

type Config struct {
	Level   encoding.LogLevel `long:"log-level"`
	Address string            `long:"address"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:   encoding.LogLevel{Level: logging.InfoLevel},
		Address: "https://ropsten.infura.io/v3/YOUR_TOKEN_HERE",
	}
}
