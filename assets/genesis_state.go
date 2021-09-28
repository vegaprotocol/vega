package assets

import (
	"encoding/json"
	"errors"

	types "code.vegaprotocol.io/protos/vega"
)

type AssetDetails struct {
	Name        string  `json:"name"`
	Symbol      string  `json:"symbol"`
	TotalSupply string  `json:"total_supply"`
	Decimals    uint64  `json:"decimals"`
	MinLpStake  string  `json:"min_lp_stake"`
	Source      *Source `json:"source"`
}

type Source struct {
	BuiltinAsset *BuiltinAsset `json:"builtin_asset,omitempty"`
	Erc20        *Erc20        `json:"erc20,omitempty"`
}

type BuiltinAsset struct {
	MaxFaucetAmountMint string `json:"max_faucet_amount_mint"`
}

type Erc20 struct {
	ContractAddress string `json:"contract_address"`
}

func (a *AssetDetails) IntoProto() (*types.AssetDetails, error) {
	if a.Source == nil || (a.Source.BuiltinAsset == nil && a.Source.Erc20 == nil) {
		return nil, errors.New("missing asset source")
	}

	if a.Source.BuiltinAsset != nil && a.Source.Erc20 != nil {
		return nil, errors.New("multiple asset sources specified")
	}

	details := types.AssetDetails{
		Name:        a.Name,
		Symbol:      a.Symbol,
		TotalSupply: a.TotalSupply,
		Decimals:    a.Decimals,
		MinLpStake:  a.MinLpStake,
	}

	if a.Source.BuiltinAsset != nil {
		details.Source = &types.AssetDetails_BuiltinAsset{
			BuiltinAsset: &types.BuiltinAsset{
				MaxFaucetAmountMint: a.Source.BuiltinAsset.MaxFaucetAmountMint,
			},
		}
	}

	if a.Source.Erc20 != nil {
		details.Source = &types.AssetDetails_Erc20{
			Erc20: &types.ERC20{
				ContractAddress: a.Source.Erc20.ContractAddress,
			},
		}
	}

	return &details, nil
}

type GenesisState map[string]AssetDetails

var (
	governanceAsset = AssetDetails{
		Name:        "VOTE",
		Symbol:      "VOTE",
		TotalSupply: "0",
		Decimals:    5,
		MinLpStake:  "1",
		Source: &Source{
			BuiltinAsset: &BuiltinAsset{
				MaxFaucetAmountMint: "10000",
			},
		},
	}

	VegaTokenMainNet = AssetDetails{
		Name:        "VEGA",
		Symbol:      "VEGA",
		TotalSupply: "64999723000000000000000000",
		Decimals:    18,
		MinLpStake:  "1",
		Source: &Source{
			Erc20: &Erc20{
				ContractAddress: "0xcB84d72e61e383767C4DFEb2d8ff7f4FB89abc6e",
			},
		},
	}

	VegaTokenTestNet = AssetDetails{
		Name:        "VEGA (V2 Test)",
		Symbol:      "VEGA",
		TotalSupply: "6499972300000",
		Decimals:    5,
		MinLpStake:  "1",
		Source: &Source{
			Erc20: &Erc20{
				ContractAddress: "0xFa521aDcc11925266A518CdE33386FcD9cF2A4A6",
			},
		},
	}
)

func DefaultGenesisState() GenesisState {
	assets := map[string]AssetDetails{
		"VOTE": governanceAsset,
	}

	return assets
}

func LoadGenesisState(bytes []byte) (map[string]*types.AssetDetails, error) {
	state := struct {
		Assets GenesisState `json:"assets"`
	}{}
	err := json.Unmarshal(bytes, &state)
	if err != nil {
		return nil, err
	}

	// now convert them all into proto
	out := map[string]*types.AssetDetails{}
	for k, v := range state.Assets {
		details, err := v.IntoProto()
		if err != nil {
			return nil, err
		}
		out[k] = details
	}

	return out, nil
}
