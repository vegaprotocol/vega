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

	CloseParameters              *closeParams
	EnactParameters              *enactParams
	DefaultMinParticipationStake uint64
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},

		CloseParameters: &closeParams{
			DefaultMinSeconds: minCloseSeconds,
			DefaultMaxSeconds: maxCloseSeconds,
		},
		EnactParameters: &enactParams{
			DefaultMinSeconds: minEnactSeconds,
			DefaultMaxSeconds: maxEnactSeconds,
		},
		DefaultMinParticipationStake: participationPercent,
	}
}
