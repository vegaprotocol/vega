package api

import (
	"time"

	"code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "api"

type Config struct {
	Level   encoding.LogLevel
	Timeout encoding.Duration

	GraphQLServerPort          int
	GraphQLServerIpAddress     string
	GraphQLSubscriptionRetries int
	RestProxyServerPort        int
	RestProxyIpAddress         string
	GrpcServerPort             int
	GrpcServerIpAddress        string
	GrpcSubscriptionRetries    int
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:   encoding.LogLevel{Level: logging.InfoLevel},
		Timeout: encoding.Duration{Duration: 5000 * time.Millisecond},

		GraphQLServerIpAddress:     "0.0.0.0",
		GraphQLServerPort:          3004,
		GraphQLSubscriptionRetries: 3,

		RestProxyIpAddress:  "0.0.0.0",
		RestProxyServerPort: 3003,

		GrpcServerIpAddress:     "0.0.0.0",
		GrpcServerPort:          3002,
		GrpcSubscriptionRetries: 3,
	}
}
