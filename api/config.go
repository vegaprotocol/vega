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
	IP               string            `long:"ip"`
	StreamRetries    int               `long:"stream-retries"`
	DisableTxCommit  bool              `long:"disable-tx-commit"`
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
		Port:             3002,
		StreamRetries:    3,
		DisableTxCommit:  true,
		CoreNodeIP:       "127.0.0.1",
		CoreNodeGRPCPort: 3004,
	}
}
