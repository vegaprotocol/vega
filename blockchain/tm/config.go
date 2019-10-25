package tm

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "tm"

type Config struct {
	Level          encoding.LogLevel
	LogTimeDebug   bool
	ClientAddr     string
	ClientEndpoint string
	ServerPort     int
	ServerAddr     string
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:          encoding.LogLevel{Level: logging.InfoLevel},
		ServerPort:     26658,
		ServerAddr:     "localhost",
		ClientAddr:     "tcp://0.0.0.0:26657",
		ClientEndpoint: "/websocket",
		LogTimeDebug:   true,
	}
}
