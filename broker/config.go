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
			DialTimeout: encoding.Duration{Duration: 2 * time.Minute},
			Timeout:     encoding.Duration{Duration: 5 * time.Second},
			IP:          "0.0.0.0",
			Port:        3005,
			Enabled:     true,
		},
	}
}

type SocketConfig struct {
	DialTimeout encoding.Duration `long:"dial-timeout" description:" "`
	Timeout     encoding.Duration `long:"timeout" description:" "`
	IP          string            `long:"ip" description:" "`
	Port        int               `long:"port" description:" "`
	Enabled     encoding.Bool     `long:"enabled" description:"Enabled socket streaming of bus events"`
}
