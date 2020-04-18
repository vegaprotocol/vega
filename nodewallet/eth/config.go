package eth

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "eth"
)

type Config struct {
	Level         encoding.LogLevel
	Address       string
	BridgeAddress string
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:         encoding.LogLevel{Level: logging.InfoLevel},
		Address:       "https://ropsten.infura.io/v3/2d4acb74430e4792b8d783fdfaa3ae82",
		BridgeAddress: "0x3EA59801698c6820328597F26d29fC3EaAa17AcA",
	}
}
