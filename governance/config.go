package governance

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "governance"

// Config represents governance specific configuration
type Config struct {
	// logging level
	Level                                                              encoding.LogLevel
	DefaultMinClose, DefaultMaxClose, DefaultMinEnact, DefaultMaxEnact int64
	DefaultMinParticipation                                            uint64
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level:                   encoding.LogLevel{Level: logging.InfoLevel},
		DefaultMinClose:         48 * 3600, // 2 days,
		DefaultMaxClose:         365 * 24 * 3600,
		DefaultMinEnact:         0,               // that's enacting from closing TS + 0 seconds
		DefaultMaxEnact:         365 * 24 * 3600, // 1 year from a proposal passing the vote
		DefaultMinParticipation: 1,
	}
}
