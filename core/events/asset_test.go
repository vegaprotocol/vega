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

package events_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/assert"
)

func TestAssetBuiltInAssetDeepClone(t *testing.T) {
	ctx := context.Background()

	a := types.Asset{
		ID: "Id",
		Details: &types.AssetDetails{
			Name:     "Name",
			Symbol:   "Symbol",
			Decimals: 5,
			Source: &types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.NewUint(100000000),
				},
			},
		},
	}

	assetEvent := events.NewAssetEvent(ctx, a)
	a2 := assetEvent.Asset()

	// Change the original and check we are not updating the wrapped event
	a.ID = "Changed"
	a.Details.Name = "Changed"
	a.Details.Symbol = "Changed"
	a.Details.Decimals = 999

	as := a.Details.Source.(*types.AssetDetailsBuiltinAsset)
	bia := as.BuiltinAsset
	bia.MaxFaucetAmountMint = num.NewUint(999)

	as2 := a2.Details.Source.(*proto.AssetDetails_BuiltinAsset)
	bia2 := as2.BuiltinAsset

	assert.NotEqual(t, a.ID, a2.Id)
	assert.NotEqual(t, a.Details.Name, a2.Details.Name)
	assert.NotEqual(t, a.Details.Symbol, a2.Details.Symbol)
	assert.NotEqual(t, a.Details.Decimals, a2.Details.Decimals)

	assert.NotEqual(t, bia.MaxFaucetAmountMint, bia2.MaxFaucetAmountMint)
}

func TestAssetERCDeepClone(t *testing.T) {
	ctx := context.Background()

	a := types.Asset{
		ID: "Id",
		Details: &types.AssetDetails{
			Name:     "Name",
			Symbol:   "Symbol",
			Decimals: 5,
			Source: &types.AssetDetailsErc20{
				ERC20: &types.ERC20{
					ContractAddress: "Contact Address",
				},
			},
		},
	}

	assetEvent := events.NewAssetEvent(ctx, a)
	a2 := assetEvent.Asset()

	// Change the original and check we are not updating the wrapped event
	a.ID = "Changed"
	a.Details.Name = "Changed"
	a.Details.Symbol = "Changed"
	a.Details.Decimals = 999

	as := a.Details.Source.(*types.AssetDetailsErc20)
	erc := as.ERC20
	erc.ContractAddress = "Changed"

	as2 := a2.Details.Source.(*proto.AssetDetails_Erc20)
	erc2 := as2.Erc20

	assert.NotEqual(t, a.ID, a2.Id)
	assert.NotEqual(t, a.Details.Name, a2.Details.Name)
	assert.NotEqual(t, a.Details.Symbol, a2.Details.Symbol)
	assert.NotEqual(t, a.Details.Decimals, a2.Details.Decimals)

	assert.NotEqual(t, erc.ContractAddress, erc2.ContractAddress)
}
