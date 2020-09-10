package assets

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "assets"
)

type Config struct {
	Level encoding.LogLevel
	ERC20 ERC20Config
}

type ERC20Config struct {
	BridgeAddress string
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig(defaultRootPath string) Config {

	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		ERC20: ERC20Config{
			BridgeAddress: "0xf6C9d3e937fb2dA4995272C1aC3f3D466B7c23fC",
		},
	}
}
