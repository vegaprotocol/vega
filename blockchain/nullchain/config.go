package nullchain

import (
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const namedLogger = "nullchain"

type Config struct {
	Level                encoding.LogLevel `long:"log-level"`
	BlockDuration        encoding.Duration `long:"block-duration" description:"(default 1s)"`
	TransactionsPerBlock uint64            `long:"transactions-per-block" description:"(default 10)"`
	GenesisFile          string            `long:"genesis-file" description:"path to a tendermint genesis file"`
	IP                   string            `long:"ip" description:"time-forwarding IP (default 26658)"`
	Port                 int               `long:"port" description:"time-forwarding port (default localhost)"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:                encoding.LogLevel{Level: logging.InfoLevel},
		BlockDuration:        encoding.Duration{Duration: 1 * time.Second},
		TransactionsPerBlock: 10,
		IP:                   "localhost",
		Port:                 26658,
	}
}
