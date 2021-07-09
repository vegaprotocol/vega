package evtforward

import (
	"time"

	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
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
	// a list of allowlisted blockchain queue public keys
	BlockchainQueueAllowlist []string `long:"blockchain-queue-allowlist" description:" "`
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level:                    encoding.LogLevel{Level: logging.InfoLevel},
		RetryRate:                encoding.Duration{Duration: defaultRetryRate},
		BlockchainQueueAllowlist: []string{},
	}
}
