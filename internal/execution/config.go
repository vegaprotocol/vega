package execution

import (
	"path/filepath"

	"code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/engines"
	"code.vegaprotocol.io/vega/internal/logging"
)

const (
	// namedLogger is the identifier for package and should ideally match the package name
	// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
	namedLogger      = "execution"
	MarketConfigPath = "markets"
)

type MarketConfig struct {
	Path    string
	Configs []string
}

type Config struct {
	Level encoding.LogLevel

	Markets MarketConfig
	Engines engines.Config
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
		Engines: engines.NewDefaultConfig(),
	}
	return c
}
