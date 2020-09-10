package assets

import (
	"encoding/json"

	types "code.vegaprotocol.io/vega/proto"
)

type GenesisState struct {
	Builtins []types.BuiltinAsset
	ERC20    []types.ERC20
}

func DefaultGenesisState() GenesisState {
	return GenesisState{
		Builtins: []types.BuiltinAsset{
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
			types.BuiltinAsset{
				Name:                "VUSD",
				Symbol:              "VUSD",
				TotalSupply:         "21000000",
				Decimals:            5,
				MaxFaucetAmountMint: "500000000", // 5000VUSD
			},
		},
		ERC20: []types.ERC20{
			{
				ContractAddress: "0x308C71DE1FdA14db838555188211Fc87ef349272",
			},
		},
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
