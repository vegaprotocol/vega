package blockchain

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "blockchain"

// Config represent the configuration of the blockchain package
type Config struct {
	Level encoding.LogLevel

	ClientAddr          string
	ClientEndpoint      string
	ServerPort          int
	ServerAddr          string
	LogTimeDebug        bool
	LogOrderSubmitDebug bool
	LogOrderAmendDebug  bool
	LogOrderCancelDebug bool
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:               encoding.LogLevel{Level: logging.InfoLevel},
		ServerPort:          26658,
		ServerAddr:          "localhost",
		ClientAddr:          "tcp://0.0.0.0:26657",
		ClientEndpoint:      "/websocket",
		LogOrderSubmitDebug: true,
		LogTimeDebug:        true,
	}
}
