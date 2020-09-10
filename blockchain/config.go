package blockchain

import (
	"code.vegaprotocol.io/vega/blockchain/noop"
	"code.vegaprotocol.io/vega/blockchain/ratelimit"
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
	RateLimit           ratelimit.Config

	Tendermint TendermintConfig
	Noop       noop.Config
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:               encoding.LogLevel{Level: logging.InfoLevel},
		LogOrderSubmitDebug: true,
		LogTimeDebug:        true,
		ChainProvider:       "tendermint",
		Tendermint:          NewDefaultTendermintConfig(),
		Noop:                noop.NewDefaultConfig(),
	}
}

type TendermintConfig struct {
	Level          encoding.LogLevel
	LogTimeDebug   bool
	ClientAddr     string
	ClientEndpoint string
	ServerPort     int
	ServerAddr     string
	RateLimit      ratelimit.Config
}

// NewDefaultTendermintConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultTendermintConfig() TendermintConfig {
	return TendermintConfig{
		Level:          encoding.LogLevel{Level: logging.InfoLevel},
		ServerPort:     26658,
		ServerAddr:     "localhost",
		ClientAddr:     "tcp://0.0.0.0:26657",
		ClientEndpoint: "/websocket",
		LogTimeDebug:   true,
	}
}
