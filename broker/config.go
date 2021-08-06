package broker

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "broker"

// Config represents the configuration of the broker.
type Config struct {
	Level  encoding.LogLevel `long:"log-level"`
	Socket SocketConfig      `group:"Socket" namespace:"socket"`
}

// NewDefaultConfig creates an instance of config with default values.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		Socket: SocketConfig{
			DialTimeout:             encoding.Duration{Duration: 2 * time.Minute},
			DialRetryInterval:       encoding.Duration{Duration: 5 * time.Second},
			SocketQueueTimeout:      encoding.Duration{Duration: 3 * time.Second},
			MaxSendTimeouts:         10,
			EventChannelBufferSize:  10000000,
			SocketChannelBufferSize: 1000000,
			IP:                      "0.0.0.0",
			Port:                    3005,
			Transport:               "tcp",
			Enabled:                 false,
		},
	}
}

type SocketConfig struct {
	DialTimeout        encoding.Duration `long:"dial-timeout" description:" "`
	DialRetryInterval  encoding.Duration `long:"dial-retry-interval" description:" "`
	SocketQueueTimeout encoding.Duration `long:"socket-queue-timeout" description:" "`

	EventChannelBufferSize  int `long:"event-channel-buffer-size" description:" "`
	SocketChannelBufferSize int `long:"socket-channel-buffer-size" description:" "`

	MaxSendTimeouts int `long:"max-send-timeouts" description:" "`

	IP        string        `long:"ip" description:"Data node IP address"`
	Port      int           `long:"port" description:"Data node port"`
	Enabled   encoding.Bool `long:"enabled" description:"Enable streaming of bus events over socket"`
	Transport string        `long:"transport" description:"Transport of socket. tcp/inproc are allowed. Default is TCP"`
}
