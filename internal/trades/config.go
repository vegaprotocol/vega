package trades

import (
	"code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "trades"

type Config struct {
	Level encoding.LogLevel

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
