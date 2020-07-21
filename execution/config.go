package execution

import (
	"path/filepath"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/positions"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"
)

const (
	// namedLogger is the identifier for package and should ideally match the package name
	// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
	namedLogger = "execution"
	// MarketConfigPath is the default path in the config folder for the market configurations
	MarketConfigPath = "markets"
)

// MarketConfig represents the configuration of the markets
type MarketConfig struct {
	Path    string
	Configs []string
}

// Config is the configuration of the execution package
type Config struct {
	Level encoding.LogLevel

	Markets                     MarketConfig
	InsurancePoolInitialBalance uint64

	Matching   matching.Config
	Risk       risk.Config
	Position   positions.Config
	Settlement settlement.Config
	Fee        fee.Config
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig(defaultConfigDirPath string) Config {
	c := Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		Markets: MarketConfig{
			Path:    filepath.Join(defaultConfigDirPath, MarketConfigPath),
			Configs: []string{},
		},
		InsurancePoolInitialBalance: 0,
		Matching:                    matching.NewDefaultConfig(),
		Risk:                        risk.NewDefaultConfig(),
		Position:                    positions.NewDefaultConfig(),
		Settlement:                  settlement.NewDefaultConfig(),
		Fee:                         fee.NewDefaultConfig(),
	}
	return c
}
