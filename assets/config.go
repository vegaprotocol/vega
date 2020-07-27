package assets

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/jsonpb"
)

const (
	namedLogger  = "assets"
	devAssetPath = "dev_assets.json"
)

type Config struct {
	Level               encoding.LogLevel
	DevAssetSourcesPath string
	ERC20               ERC20Config
}

type ERC20Config struct {
	BridgeAddress string
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig(defaultRootPath string) Config {

	return Config{
		Level:               encoding.LogLevel{Level: logging.InfoLevel},
		DevAssetSourcesPath: filepath.Join(defaultRootPath, devAssetPath),
		ERC20: ERC20Config{
			BridgeAddress: "0xf6C9d3e937fb2dA4995272C1aC3f3D466B7c23fC",
		},
	}
}

func GenDevAssetSourcesPath(defaultRootPath string) error {
	assets := types.DevAssets{Sources: []*types.AssetSource{
		&types.AssetSource{
			Source: &types.AssetSource_BuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					Name:        "VegaToken",
					Symbol:      "VGT",
					TotalSupply: "10000000",
					Decimals:    5,
				},
			},
		},
		&types.AssetSource{
			Source: &types.AssetSource_BuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					Name:        "Ether",
					Symbol:      "ETH",
					TotalSupply: "110436690",
					Decimals:    5,
				},
			},
		},
		&types.AssetSource{
			Source: &types.AssetSource_BuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					Name:        "Bitcoin",
					Symbol:      "BTC",
					TotalSupply: "21000000",
					Decimals:    5,
				},
			},
		},
		&types.AssetSource{
			Source: &types.AssetSource_BuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					Name:        "VUSD",
					Symbol:      "VUSD",
					TotalSupply: "21000000",
					Decimals:    5,
				},
			},
		},
		// this is the VUSD
		// &types.AssetSource{
		// 	Source: &types.AssetSource_Erc20{
		// 		Erc20: &types.ERC20{
		// 			ContractAddress: "0x955C6789A7fbee203B4bE0F01428E769308813f2",
		// 		},
		// 	},
		// },
	}}

	m := jsonpb.Marshaler{
		Indent: "  ",
	}
	buf, err := m.MarshalToString(&assets)
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(defaultRootPath, devAssetPath))
	if err != nil {
		return err
	}

	if _, err = f.WriteString(string(buf)); err != nil {
		return err
	}
	return nil
}

func LoadDevAssets(cfg Config) ([]*types.AssetSource, error) {
	path := filepath.Join(cfg.DevAssetSourcesPath)
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	assets := types.DevAssets{}
	err = jsonpb.Unmarshal(strings.NewReader(string(buf)), &assets)
	if err != nil {
		return nil, err
	}
	return assets.Sources, nil
}
