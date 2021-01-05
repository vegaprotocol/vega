package assets

import (
	"encoding/hex"
	"encoding/json"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
	"golang.org/x/crypto/sha3"
)

type GenesisState struct {
	Builtins map[string]types.BuiltinAsset
	ERC20    map[string]types.ERC20
}

var (
	governanceAsset = types.BuiltinAsset{
		Name:                "VOTE",
		Symbol:              "VOTE",
		TotalSupply:         "0",
		Decimals:            5,
		MaxFaucetAmountMint: "100000",
	}

	defaultBuiltins = []types.BuiltinAsset{
		{
			Name:                "Ether",
			Symbol:              "ETH",
			TotalSupply:         "110436690",
			Decimals:            5,
			MaxFaucetAmountMint: "10000000", // 100ETH
		},
		{
			Name:                "Bitcoin",
			Symbol:              "BTC",
			TotalSupply:         "21000000",
			Decimals:            5,
			MaxFaucetAmountMint: "1000000", // 10BTC
		},
		{
			Name:                "VUSD",
			Symbol:              "VUSD",
			TotalSupply:         "21000000",
			Decimals:            5,
			MaxFaucetAmountMint: "500000000", // 5000VUSD
		},
	}

	defaultERC20s = []types.ERC20{
		{
			ContractAddress: "0x308C71DE1FdA14db838555188211Fc87ef349272",
		},
	}
)

func DefaultGenesisState() GenesisState {
	builtins := make(map[string]types.BuiltinAsset, len(defaultBuiltins))
	erc20s := make(map[string]types.ERC20, len(defaultERC20s))

	h := func(key []byte) []byte {
		hasher := sha3.New256()
		hasher.Write([]byte(key))
		return hasher.Sum(nil)
	}

	builtins["VOTE"] = governanceAsset

	for _, v := range defaultBuiltins {
		assetSrc := types.AssetSource{
			Source: &types.AssetSource_BuiltinAsset{
				BuiltinAsset: &v,
			},
		}
		builtins[hex.EncodeToString(h([]byte(assetSrc.String())))] = v
	}

	for _, v := range defaultERC20s {
		assetSrc := types.AssetSource{
			Source: &types.AssetSource_Erc20{
				Erc20: &v,
			},
		}
		erc20s[hex.EncodeToString(h([]byte(assetSrc.String())))] = v
	}

	return GenesisState{
		Builtins: builtins,
		ERC20:    erc20s,
	}
}

func LoadGenesisState(bytes []byte) (*GenesisState, error) {
	state := struct {
		Assets *GenesisState `json:"assets"`
	}{}
	err := json.Unmarshal(bytes, &state)
	if err != nil {
		return nil, err
	}
	return state.Assets, nil
}
