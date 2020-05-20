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

	// this split allows partially setting network parameters
	//CloseParameters              *closeParams
	//EnactParameters              *enactParams
	//DefaultMinParticipationStake uint64
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
	}
}
