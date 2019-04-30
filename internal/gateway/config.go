package gateway

import (
	"time"

	"code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
)

type ServerConfig struct {
	Port int
	IP   string
}

type GatewayServiceConfig struct {
	ServerConfig
	Enabled bool
}

type Config struct {
	Level               encoding.LogLevel
	Timeout             encoding.Duration
	Node                ServerConfig
	GraphQL             GatewayServiceConfig
	REST                GatewayServiceConfig
	SubscriptionRetries int
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:   encoding.LogLevel{Level: logging.InfoLevel},
		Timeout: encoding.Duration{Duration: 5 * time.Second},
		GraphQL: GatewayServiceConfig{
			ServerConfig: ServerConfig{
				IP:   "0.0.0.0",
				Port: 3004,
			},
			Enabled: true,
		},
		REST: GatewayServiceConfig{
			ServerConfig: ServerConfig{
				IP:   "0.0.0.0",
				Port: 3003,
			},
			Enabled: true,
		},
		Node: ServerConfig{
			IP:   "0.0.0.0",
			Port: 3002,
		},
		SubscriptionRetries: 3,
	}
}
