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

package events_test

import (
	"context"
	"testing"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

func TestAssetBuiltInAssetDeepClone(t *testing.T) {
	ctx := context.Background()

	a := types.Asset{
		ID: "Id",
		Details: &types.AssetDetails{
			Name:        "Name",
			Symbol:      "Symbol",
			TotalSupply: num.NewUint(10000),
			Decimals:    5,
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
	a.Details.TotalSupply = num.NewUint(999)
	a.Details.Decimals = 999

	as := a.Details.Source.(*types.AssetDetailsBuiltinAsset)
	bia := as.BuiltinAsset
	bia.MaxFaucetAmountMint = num.NewUint(999)

	as2 := a2.Details.Source.(*proto.AssetDetails_BuiltinAsset)
	bia2 := as2.BuiltinAsset

	assert.NotEqual(t, a.ID, a2.Id)
	assert.NotEqual(t, a.Details.Name, a2.Details.Name)
	assert.NotEqual(t, a.Details.Symbol, a2.Details.Symbol)
	assert.NotEqual(t, a.Details.TotalSupply, a2.Details.TotalSupply)
	assert.NotEqual(t, a.Details.Decimals, a2.Details.Decimals)

	assert.NotEqual(t, bia.MaxFaucetAmountMint, bia2.MaxFaucetAmountMint)
}

func TestAssetERCDeepClone(t *testing.T) {
	ctx := context.Background()

	a := types.Asset{
		ID: "Id",
		Details: &types.AssetDetails{
			Name:        "Name",
			Symbol:      "Symbol",
			TotalSupply: num.NewUint(10000),
			Decimals:    5,
			Source: &types.AssetDetailsErc20{
				Erc20: &types.ERC20{
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
	a.Details.TotalSupply = num.NewUint(999)
	a.Details.Decimals = 999

	as := a.Details.Source.(*types.AssetDetailsErc20)
	erc := as.Erc20
	erc.ContractAddress = "Changed"

	as2 := a2.Details.Source.(*proto.AssetDetails_Erc20)
	erc2 := as2.Erc20

	assert.NotEqual(t, a.ID, a2.Id)
	assert.NotEqual(t, a.Details.Name, a2.Details.Name)
	assert.NotEqual(t, a.Details.Symbol, a2.Details.Symbol)
	assert.NotEqual(t, a.Details.TotalSupply, a2.Details.TotalSupply)
	assert.NotEqual(t, a.Details.Decimals, a2.Details.Decimals)

	assert.NotEqual(t, erc.ContractAddress, erc2.ContractAddress)
}
