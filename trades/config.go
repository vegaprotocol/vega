package trades

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "trades"

// Config represent the configuration of the trades service
type Config struct {
	Level encoding.LogLevel `long:"log-level"`

	// PageSizeDefault sets the default page size
	PageSizeDefault uint64

	// PageSizeMaximum sets the maximum page size
	PageSizeMaximum uint64
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:           encoding.LogLevel{Level: logging.InfoLevel},
		PageSizeDefault: 200,
		PageSizeMaximum: 200,
	}
}
