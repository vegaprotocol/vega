package evtforward

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	// how often the evtforward needs to select a node to
	// send the event if nothing was received
	defaultRetryRate = 10 * time.Second
)

// Config represents governance specific configuration
type Config struct {
	// logging level
	Level     encoding.LogLevel `long:"log-level"`
	RetryRate encoding.Duration `long:"retry-rate"`
	// a list of whitelisted blockchain queue public keys
	BlockchainQueueWhitelist []string `long:"blockchain-queue-whitelist" description:" "`
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level:                    encoding.LogLevel{Level: logging.InfoLevel},
		RetryRate:                encoding.Duration{Duration: defaultRetryRate},
		BlockchainQueueWhitelist: []string{},
	}
}
