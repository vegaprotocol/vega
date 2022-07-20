// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
	Quantum     string  `json:"quantum"`
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
	ContractAddress   string `json:"contract_address"`
	LifetimeLimit     string `json:"lifetime_limit"`
	WithdrawThreshold string `json:"withdraw_threshold"`
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
		Quantum:     a.Quantum,
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
				ContractAddress:   a.Source.Erc20.ContractAddress,
				WithdrawThreshold: a.Source.Erc20.WithdrawThreshold,
				LifetimeLimit:     a.Source.Erc20.LifetimeLimit,
			},
		}
	}

	return &details, nil
}

type GenesisState map[string]AssetDetails

var governanceAsset = AssetDetails{
	Name:        "VOTE",
	Symbol:      "VOTE",
	TotalSupply: "0",
	Decimals:    5,
	Quantum:     "1",
	Source: &Source{
		BuiltinAsset: &BuiltinAsset{
			MaxFaucetAmountMint: "10000",
		},
	},
}

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
