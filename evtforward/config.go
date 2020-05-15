package evtforward

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "evtforward"

	// how often the evtforward needs to select a node to
	// send the event if nothing was received
	defaultRetryRate = 10 * time.Second
)

// Config represents governance specific configuration
type Config struct {
	// logging level
	Level     encoding.LogLevel
	RetryRate encoding.Duration
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level:     encoding.LogLevel{Level: logging.InfoLevel},
		RetryRate: encoding.Duration{Duration: defaultRetryRate},
	}
}
