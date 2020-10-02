package gateway

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// ServerConfig represent the configuration of a server in vega
type ServerConfig struct {
	Port int    `long:"port" description:"Listen for connection on port <port>"`
	IP   string `long:"ip" description:"Bind to address <IP>"`
}

// GraphqlServiceConfig represents the configuration of the gateway
type GraphqlServiceConfig struct {
	ServerConfig
	Enabled         bool `long:"enabled"`
	ComplexityLimit int  `long:"complexity-limit"`
}

// RESTGatewayServiceConfig represent the configuration of the rest service
type RESTGatewayServiceConfig struct {
	ServerConfig
	Enabled    bool `long:"enabled"`
	APMEnabled bool
}

// Config represents the general configuration for the gateway
type Config struct {
	Level                    encoding.LogLevel        `long:"log-level" choice:"debug" choice:"info" choice:"warning"`
	Timeout                  encoding.Duration        `long:"timeout"`
	Node                     ServerConfig             `group:"Node" namespace:"node"`
	GraphQL                  GraphqlServiceConfig     `group:"GraphQL" namespace:"graphql"`
	REST                     RESTGatewayServiceConfig `group:"REST" namespace:"rest"`
	SubscriptionRetries      int                      `long:"subscription-retries"`
	GraphQLPlaygroundEnabled bool                     `long:"graphql-playground" description:"Enables the GraphQL playground"`
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
			Enabled: true,
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
