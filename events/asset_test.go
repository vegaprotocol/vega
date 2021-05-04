package events_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/proto"
	"github.com/stretchr/testify/assert"
)

func TestAssetBuiltInAssetDeepClone(t *testing.T) {
	ctx := context.Background()

	a := proto.Asset{
		Id: "Id",
		Details: &proto.AssetDetails{
			Name:        "Name",
			Symbol:      "Symbol",
			TotalSupply: "10000",
			Decimals:    5,
			Source: &proto.AssetDetails_BuiltinAsset{
				BuiltinAsset: &proto.BuiltinAsset{
					MaxFaucetAmountMint: "100000000",
				},
			},
		},
	}

	assetEvent := events.NewAssetEvent(ctx, a)
	a2 := assetEvent.Asset()

	// Change the original and check we are not updating the wrapped event
	a.Id = "Changed"
	a.Details.Name = "Changed"
	a.Details.Symbol = "Changed"
	a.Details.TotalSupply = "999"
	a.Details.Decimals = 999

	as := a.Details.Source.(*proto.AssetDetails_BuiltinAsset)
	bia := as.BuiltinAsset
	bia.MaxFaucetAmountMint = "999"

	as2 := a2.Details.Source.(*proto.AssetDetails_BuiltinAsset)
	bia2 := as2.BuiltinAsset

	assert.NotEqual(t, a.Id, a2.Id)
	assert.NotEqual(t, a.Details.Name, a2.Details.Name)
	assert.NotEqual(t, a.Details.Symbol, a2.Details.Symbol)
	assert.NotEqual(t, a.Details.TotalSupply, a2.Details.TotalSupply)
	assert.NotEqual(t, a.Details.Decimals, a2.Details.Decimals)

	assert.NotEqual(t, bia.MaxFaucetAmountMint, bia2.MaxFaucetAmountMint)
}

func TestAssetERCDeepClone(t *testing.T) {
	ctx := context.Background()

	a := proto.Asset{
		Id: "Id",
		Details: &proto.AssetDetails{
			Name:        "Name",
			Symbol:      "Symbol",
			TotalSupply: "10000",
			Decimals:    5,
			Source: &proto.AssetDetails_Erc20{
				Erc20: &proto.ERC20{
					ContractAddress: "Contact Address",
				},
			},
		},
	}

	assetEvent := events.NewAssetEvent(ctx, a)
	a2 := assetEvent.Asset()

	// Change the original and check we are not updating the wrapped event
	a.Id = "Changed"
	a.Details.Name = "Changed"
	a.Details.Symbol = "Changed"
	a.Details.TotalSupply = "999"
	a.Details.Decimals = 999

	as := a.Details.Source.(*proto.AssetDetails_Erc20)
	erc := as.Erc20
	erc.ContractAddress = "Changed"

	as2 := a2.Details.Source.(*proto.AssetDetails_Erc20)
	erc2 := as2.Erc20

	assert.NotEqual(t, a.Id, a2.Id)
	assert.NotEqual(t, a.Details.Name, a2.Details.Name)
	assert.NotEqual(t, a.Details.Symbol, a2.Details.Symbol)
	assert.NotEqual(t, a.Details.TotalSupply, a2.Details.TotalSupply)
	assert.NotEqual(t, a.Details.Decimals, a2.Details.Decimals)

	assert.NotEqual(t, erc.ContractAddress, erc2.ContractAddress)
}
