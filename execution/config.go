package execution

import (
	"path/filepath"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/liquidity"
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
	Path    string   `long:"path"`
	Configs []string `long:"configs"`
}

// Config is the configuration of the execution package
type Config struct {
	Level encoding.LogLevel `long:"log-level"`

	Markets MarketConfig `group:"Markets" namespace:"markets"`

	Matching   matching.Config   `group:"Matching" namespace:"matching"`
	Risk       risk.Config       `group:"Risk" namespace:"risk"`
	Position   positions.Config  `group:"Position" namespace:"position"`
	Settlement settlement.Config `group:"Settlement" namespace:"settlement"`
	Fee        fee.Config        `group:"Fee" namespace:"fee"`
	Liquidity  liquidity.Config  `group:"Liquidity" namespace:"liquidity"`
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
		Matching:   matching.NewDefaultConfig(),
		Risk:       risk.NewDefaultConfig(),
		Position:   positions.NewDefaultConfig(),
		Settlement: settlement.NewDefaultConfig(),
		Fee:        fee.NewDefaultConfig(),
		Liquidity:  liquidity.NewDefaultConfig(),
	}
	return c
}
