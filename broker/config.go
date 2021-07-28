package broker

import (
	"time"

	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
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
			IP:            "0.0.0.0",
			Port:          3005,
			MaxRetries:    10,
			RetryInternal: encoding.Duration{Duration: 50 * time.Millisecond},
		},
	}
}

type SocketConfig struct {
	IP            string            `long:"ip" description:" "`
	Port          int               `long:"port" description:" "`
	MaxRetries    int               `long:"max-retries"`
	RetryInternal encoding.Duration `long:"retry-interval"`
}
