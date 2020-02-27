package governance

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "governance"

// Config represents governance specific configuration
type Config struct {
	// logging level
	Level encoding.LogLevel
	// enable governance functionality
	Enabled bool

	MinCloseInDays uint64
	MaxCloseInDays uint64

	MinEnactInDays        uint64
	MaxEnactInDays        uint64
	MinParticipationStake uint64
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level:   encoding.LogLevel{Level: logging.InfoLevel},
		Enabled: true,

		MinCloseInDays: 1,
		MaxCloseInDays: 365,

		MinEnactInDays:        1,
		MaxEnactInDays:        365,
		MinParticipationStake: 1,
	}
}
