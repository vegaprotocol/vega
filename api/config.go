package api

import (
	"time"

	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "api.grpc"

// Config represents the configuration of the api package
type Config struct {
	Level            encoding.LogLevel `long:"log-level"`
	Timeout          encoding.Duration `long:"timeout"`
	Port             int               `long:"port"`
	WebUIPort        int               `long:"web-ui-port"`
	WebUIEnabled     encoding.Bool     `long:"web-ui-enabled"`
	Reflection       encoding.Bool     `long:"reflection"`
	IP               string            `long:"ip"`
	StreamRetries    int               `long:"stream-retries"`
	CoreNodeIP       string            `long:"core-node-ip"`
	CoreNodeGRPCPort int               `long:"core-node-grpc-port"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:   encoding.LogLevel{Level: logging.InfoLevel},
		Timeout: encoding.Duration{Duration: 5000 * time.Millisecond},

		IP:               "0.0.0.0",
		Port:             3007,
		WebUIPort:        3006,
		WebUIEnabled:     false,
		Reflection:       false,
		StreamRetries:    3,
		CoreNodeIP:       "127.0.0.1",
		CoreNodeGRPCPort: 3002,
	}
}
