package auth

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "auth"

type Config struct {
	Level      encoding.LogLevel
	Timeout    encoding.Duration
	ServerAddr string
	Enabled    bool
	Interval   encoding.Duration
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:      encoding.LogLevel{Level: logging.InfoLevel},
		Timeout:    encoding.Duration{Duration: 5000 * time.Millisecond},
		ServerAddr: "http://localhost",
		Enabled:    false,
		Interval:   encoding.Duration{Duration: 10 * time.Second},
	}
}
