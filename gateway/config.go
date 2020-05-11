package gateway

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// ServerConfig represent the configuration of a server in vega
type ServerConfig struct {
	Port int
	IP   string
}

// GraphqlServiceConfig represents the configuration of the gateway
type GraphqlServiceConfig struct {
	ServerConfig
	Enabled         bool
	ComplexityLimit int
}

// RESTGatewayServiceConfig represent the configuration of the rest service
type RESTGatewayServiceConfig struct {
	ServerConfig
	Enabled    bool
	APMEnabled bool
}

// Config represents the general configuration for the gateway
type Config struct {
	Level                    encoding.LogLevel
	Timeout                  encoding.Duration
	Node                     ServerConfig
	GraphQL                  GraphqlServiceConfig
	REST                     RESTGatewayServiceConfig
	SubscriptionRetries      int
	GraphQLPlaygroundEnabled bool
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:   encoding.LogLevel{Level: logging.InfoLevel},
		Timeout: encoding.Duration{Duration: 5 * time.Second},
		GraphQL: GraphqlServiceConfig{
			ServerConfig: ServerConfig{
				IP:   "0.0.0.0",
				Port: 3004,
			},
			Enabled:         true,
			ComplexityLimit: 5,
		},
		REST: RESTGatewayServiceConfig{
			ServerConfig: ServerConfig{
				IP:   "0.0.0.0",
				Port: 3003,
			},
			Enabled:    true,
			APMEnabled: true,
		},
		Node: ServerConfig{
			IP:   "0.0.0.0",
			Port: 3002,
		},
		SubscriptionRetries:      3,
		GraphQLPlaygroundEnabled: true,
	}
}
