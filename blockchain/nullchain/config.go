package nullchain

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "nullchain"

type Config struct {
	Level                encoding.LogLevel `long:"log-level"`
	BlockDuration        encoding.Duration `long:"block-duration"`
	TransactionsPerBlock uint64            `long:"transactions-per-block"`
	GenesisFile          string            `long:"genesis-file"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:                encoding.LogLevel{Level: logging.InfoLevel},
		BlockDuration:        encoding.Duration{Duration: 1 * time.Second},
		TransactionsPerBlock: 10,
	}
}
