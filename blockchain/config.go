package blockchain

import (
	"code.vegaprotocol.io/vega/blockchain/noop"
	"code.vegaprotocol.io/vega/blockchain/tm"
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "blockchain"

// Config represent the configuration of the blockchain package
type Config struct {
	Level               encoding.LogLevel
	LogTimeDebug        bool
	LogOrderSubmitDebug bool
	LogOrderAmendDebug  bool
	LogOrderCancelDebug bool
	ChainProvider       string

	Tendermint tm.Config
	Noop       noop.Config
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:               encoding.LogLevel{Level: logging.InfoLevel},
		LogOrderSubmitDebug: true,
		LogTimeDebug:        true,
		ChainProvider:       "noop",
		Tendermint:          tm.NewDefaultConfig(),
		Noop:                noop.NewDefaultConfig(),
	}
}
