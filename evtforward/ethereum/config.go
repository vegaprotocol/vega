package ethereum

import "code.vegaprotocol.io/vega/config/encoding"

type Config struct {
	// Level specifies the logging level of the Ethereum implementation of the
	// Event Forwarder.
	Level encoding.LogLevel `long:"log-level"`
}
