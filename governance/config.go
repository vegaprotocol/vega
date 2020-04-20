package governance

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "governance"

	// default param values - constant, and not part of the config itself
	minClose             = 48 * 3600       // 2 days
	maxClose             = 365 * 24 * 3600 // 1 year
	minEnact             = 0               // 0 -> >= close value, this has no real use
	maxEnact             = 365 * 24 * 3600 // actually same as minEnact, but there is an upper limt
	participationPercent = 1               // percentage!
)

type params struct {
	DefaultMinClose, DefaultMaxClose, DefaultMinEnact, DefaultMaxEnact int64
	DefaultMinParticipation                                            uint64
}

// Config represents governance specific configuration
type Config struct {
	// logging level
	Level  encoding.LogLevel
	params params // not exported because it's not part of the serialised config
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		params: params{
			DefaultMinClose:         minClose,
			DefaultMaxClose:         maxClose,
			DefaultMinEnact:         minEnact,
			DefaultMaxEnact:         maxEnact,
			DefaultMinParticipation: participationPercent,
		},
	}
}

func (c *Config) initParams() {
	c.params = params{
		DefaultMinClose:         minClose,
		DefaultMaxClose:         maxClose,
		DefaultMinEnact:         minEnact,
		DefaultMaxEnact:         maxEnact,
		DefaultMinParticipation: participationPercent,
	}
}
