package nodewallet

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet/eth"
)

type Config struct {
	Level    encoding.LogLevel `long:"log-level"`
	ETH      eth.Config        `group:"ETH" namespace:"eth"`
}

// NewDefaultConfig creates an instance of the package specific configuration,
// given a pointer to a logger instance to be used for logging within the
// package.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		ETH:   eth.NewDefaultConfig(),
	}
}
