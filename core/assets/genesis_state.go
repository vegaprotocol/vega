// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package assets

import (
	"encoding/json"
	"errors"

	types "code.vegaprotocol.io/vega/protos/vega"
)

type AssetDetails struct {
	Name     string  `json:"name"`
	Symbol   string  `json:"symbol"`
	Decimals uint64  `json:"decimals"`
	Quantum  string  `json:"quantum"`
	Source   *Source `json:"source"`
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
	ChainID           string `json:"chain_id"`
}

func (a *AssetDetails) IntoProto() (*types.AssetDetails, error) {
	if a.Source == nil || (a.Source.BuiltinAsset == nil && a.Source.Erc20 == nil) {
		return nil, errors.New("missing asset source")
	}

	if a.Source.BuiltinAsset != nil && a.Source.Erc20 != nil {
		return nil, errors.New("multiple asset sources specified")
	}

	details := types.AssetDetails{
		Name:     a.Name,
		Symbol:   a.Symbol,
		Decimals: a.Decimals,
		Quantum:  a.Quantum,
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
				ChainId:           a.Source.Erc20.ChainID,
			},
		}
	}

	return &details, nil
}

type GenesisState map[string]AssetDetails

var governanceAsset = AssetDetails{
	Name:     "VOTE",
	Symbol:   "VOTE",
	Decimals: 5,
	Quantum:  "1",
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
