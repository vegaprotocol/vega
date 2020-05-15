package governance

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "governance"

type closeParams struct {
	DefaultMinSeconds int64
	DefaultMaxSeconds int64
}
type enactParams struct {
	DefaultMinSeconds int64
	DefaultMaxSeconds int64
}

// Config represents governance specific configuration
type Config struct {
	// logging level
	Level encoding.LogLevel

	// this split allows partially setting network parameters
	CloseParameters              *closeParams
	EnactParameters              *enactParams
	DefaultMinParticipationStake uint64
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	defaults := defaultNetworkParameters()
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},

		CloseParameters: &closeParams{
			// time.Duration is in nanoseconds hence conversion
			DefaultMinSeconds: int64(defaults.minClose.Seconds()),
			DefaultMaxSeconds: int64(defaults.maxClose.Seconds()),
		},
		EnactParameters: &enactParams{
			DefaultMinSeconds: int64(defaults.minEnact.Seconds()),
			DefaultMaxSeconds: int64(defaults.maxEnact.Seconds()),
		},
		DefaultMinParticipationStake: defaults.minParticipationStake,
	}
}
