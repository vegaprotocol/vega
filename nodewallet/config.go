package nodewallet

import (
	"path/filepath"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet/eth"
)

const (
	namedLogger      = "nodewallet"
	defaultStoreFile = "nodewalletstore"
	devWalletsFolder = "node_wallets_dev"
)

type Config struct {
	Level          encoding.LogLevel `long:"level"`
	StorePath      string            `long:"store-path"`
	DevWalletsPath string            `long:"dev-wallets-path"`
	ETH            eth.Config        `group:"ETH" namespace:"eth"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig(defaultDirPath string) Config {
	return Config{
		Level:          encoding.LogLevel{Level: logging.InfoLevel},
		StorePath:      filepath.Join(defaultDirPath, defaultStoreFile),
		DevWalletsPath: filepath.Join(defaultDirPath, devWalletsFolder),
		ETH:            eth.NewDefaultConfig(),
	}
}
