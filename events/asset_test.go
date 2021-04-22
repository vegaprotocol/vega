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
		Id:          "Id",
		Name:        "Name",
		Symbol:      "Symbol",
		TotalSupply: "10000",
		Decimals:    5,
		Source: &proto.AssetSource{
			Source: &proto.AssetSource_BuiltinAsset{
				BuiltinAsset: &proto.BuiltinAsset{
					Name:                "Name",
					Symbol:              "Symbol",
					TotalSupply:         "100000",
					Decimals:            5,
					MaxFaucetAmountMint: "100000000",
				},
			},
		},
	}

	assetEvent := events.NewAssetEvent(ctx, a)
	a2 := assetEvent.Asset()

	// Change the original and check we are not updating the wrapped event
	a.Id = "Changed"
	a.Name = "Changed"
	a.Symbol = "Changed"
	a.TotalSupply = "999"
	a.Decimals = 999

	as := a.Source.Source.(*proto.AssetSource_BuiltinAsset)
	bia := as.BuiltinAsset
	bia.Name = "Changed"
	bia.Symbol = "Changed"
	bia.TotalSupply = "999"
	bia.Decimals = 999
	bia.MaxFaucetAmountMint = "999"

	as2 := a2.Source.Source.(*proto.AssetSource_BuiltinAsset)
	bia2 := as2.BuiltinAsset

	assert.NotEqual(t, a.Id, a2.Id)
	assert.NotEqual(t, a.Name, a2.Name)
	assert.NotEqual(t, a.Symbol, a2.Symbol)
	assert.NotEqual(t, a.TotalSupply, a2.TotalSupply)
	assert.NotEqual(t, a.Decimals, a2.Decimals)

	assert.NotEqual(t, bia.Name, bia2.Name)
	assert.NotEqual(t, bia.Symbol, bia2.Symbol)
	assert.NotEqual(t, bia.TotalSupply, bia2.TotalSupply)
	assert.NotEqual(t, bia.Decimals, bia2.Decimals)
	assert.NotEqual(t, bia.MaxFaucetAmountMint, bia2.MaxFaucetAmountMint)
}

func TestAssetERCDeepClone(t *testing.T) {
	ctx := context.Background()

	a := proto.Asset{
		Id:          "Id",
		Name:        "Name",
		Symbol:      "Symbol",
		TotalSupply: "10000",
		Decimals:    5,
		Source: &proto.AssetSource{
			Source: &proto.AssetSource_Erc20{
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
	a.Name = "Changed"
	a.Symbol = "Changed"
	a.TotalSupply = "999"
	a.Decimals = 999

	as := a.Source.Source.(*proto.AssetSource_Erc20)
	erc := as.Erc20
	erc.ContractAddress = "Changed"

	as2 := a2.Source.Source.(*proto.AssetSource_Erc20)
	erc2 := as2.Erc20

	assert.NotEqual(t, a.Id, a2.Id)
	assert.NotEqual(t, a.Name, a2.Name)
	assert.NotEqual(t, a.Symbol, a2.Symbol)
	assert.NotEqual(t, a.TotalSupply, a2.TotalSupply)
	assert.NotEqual(t, a.Decimals, a2.Decimals)

	assert.NotEqual(t, erc.ContractAddress, erc2.ContractAddress)
}
