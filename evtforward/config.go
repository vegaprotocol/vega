package evtforward

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/evtforward/ethereum"
	"code.vegaprotocol.io/vega/logging"
)

const (
	forwarderLogger = "forwarder"
	// how often the Event Forwarder needs to select a node to send the event
	// if nothing was received.
	defaultRetryRate = 10 * time.Second
)

// Config represents governance specific configuration.
type Config struct {
	// Level specifies the logging level of the Event Forwarder engine.
	Level     encoding.LogLevel `long:"log-level"`
	RetryRate encoding.Duration `long:"retry-rate"`
	// a list of allowlisted blockchain queue public keys
	BlockchainQueueAllowlist []string `long:"blockchain-queue-allowlist" description:" "`
	// Ethereum groups the configuration related to Ethereum implementation of
	// the Event Forwarder.
	Ethereum ethereum.Config `group:"Ethereum" namespace:"ethereum"`
}

// NewDefaultConfig creates an instance of the package specific configuration.
func NewDefaultConfig() Config {
	return Config{
		Level:                    encoding.LogLevel{Level: logging.InfoLevel},
		RetryRate:                encoding.Duration{Duration: defaultRetryRate},
		BlockchainQueueAllowlist: []string{},
		Ethereum: ethereum.Config{
			Level: encoding.LogLevel{Level: logging.InfoLevel},
		},
	}
}
