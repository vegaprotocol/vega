package collat

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "collateral"

// Config represent the configuration of the collateral engine
type Config struct {
	Level encoding.LogLevel
	// Auto-create trader accounts if needed?
	CreateTraderAccounts        bool
	TraderGeneralAccountBalance int64
	TraderMarginPercent         int64 // 1 for 1%, will take TraderGeneralAccountBalance/100 * TraderMarginPercent
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:                       encoding.LogLevel{Level: logging.InfoLevel},
		CreateTraderAccounts:        true,
		TraderGeneralAccountBalance: 100000000,
		TraderMarginPercent:         1,
	}
}
