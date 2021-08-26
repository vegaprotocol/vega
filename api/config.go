package api

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "api.grpc"

// Config represents the configuration of the api package
type Config struct {
	Level           encoding.LogLevel `long:"log-level"`
	Timeout         encoding.Duration `long:"timeout"`
	Port            int               `long:"port"`
	IP              string            `long:"ip"`
	StreamRetries   int               `long:"stream-retries"`
	DisableTxCommit bool              `long:"disable-tx-commit"`
	REST            RESTServiceConfig `group:"REST" namespace:"rest"`
}

// RESTGatewayServiceConfig represent the configuration of the rest service
type RESTServiceConfig struct {
	Port       int           `long:"port" description:"Listen for connection on port <port>"`
	IP         string        `long:"ip" description:"Bind to address <ip>"`
	Enabled    encoding.Bool `long:"enabled" choice:"true"  description:"Start the REST gateway"`
	APMEnabled encoding.Bool `long:"apm-enabled" choice:"true"  description:" "`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:   encoding.LogLevel{Level: logging.InfoLevel},
		Timeout: encoding.Duration{Duration: 5000 * time.Millisecond},

		IP:              "0.0.0.0",
		Port:            3002,
		StreamRetries:   3,
		DisableTxCommit: true,
		REST: RESTServiceConfig{
			IP:         "0.0.0.0",
			Port:       3003,
			Enabled:    true,
			APMEnabled: true,
		},
	}
}
