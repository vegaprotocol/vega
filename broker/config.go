package broker

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "broker"

// Config represents the configuration of the broker.
type Config struct {
	Level        encoding.LogLevel `long:"log-level"`
	SocketConfig SocketConfig      `group:"Socket" namespace:"socket"`
}

// NewDefaultConfig creates an instance of config with default values.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		SocketConfig: SocketConfig{
			DialTimeout:             encoding.Duration{Duration: 2 * time.Minute},
			DialRetryInterval:       encoding.Duration{Duration: 5 * time.Second},
			EventChannelBufferSize:  10000000,
			SocketChannelBufferSize: 1000000,
			IP:                      "0.0.0.0",
			Port:                    3005,
			Enabled:                 false,
		},
	}
}

type SocketConfig struct {
	DialTimeout       encoding.Duration `long:"dial-timeout" description:" "`
	DialRetryInterval encoding.Duration `long:"dial-retry-interval" description:" "`

	EventChannelBufferSize  int `long:"event-channel-buffer-size" description:" "`
	SocketChannelBufferSize int `long:"socket-channel-buffer-size" description:" "`

	IP      string        `long:"ip" description:" "`
	Port    int           `long:"port" description:" "`
	Enabled encoding.Bool `long:"enabled" description:"Enable streaming of bus events over socket"`
}
