package assets

import (
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

const (
	namedLogger = "assets"
)

type Config struct {
	Level      encoding.LogLevel
	TestAssets []types.Asset
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig(defaultDirPath string) Config {
	return Config{
		Level: encoding.LogLevel{Level: logging.InfoLevel},
		TestAssets: []types.Asset{
			types.Asset{
				Id:          1,
				Name:        "VegaToken",
				Symbol:      "VGT",
				TotalSupply: 42,
				Decimals:    5,
				Source: &types.Asset_BuiltinAsset{
					BuiltinAsset: &types.BuiltinAsset{
						Name:        "VegaToken",
						Symbol:      "VGT",
						TotalSupply: 42,
						Decimals:    5,
					},
				},
			},
		},
	}
}
