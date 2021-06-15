//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	pb "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types/num"
)

// Asset is the Vega representation of an external asset
type Asset struct {
	// Internal identifier of the asset
	Id string
	// Name of the asset (e.g: Great British Pound)
	Name string
	// Symbol of the asset (e.g: GBP)
	Symbol string
	// Total circulating supply for the asset
	TotalSupply *num.Uint
	// Number of decimals / precision handled by this asset
	Decimals uint8
	// The definition of the external source for this asset
	Source *AssetSource
}

// DeepClone returns a deep clone of a.
func (a Asset) DeepClone() *Asset {
	cpy := a
	if a.TotalSupply != nil {
		cpy.TotalSupply = a.TotalSupply.Clone()
	}
	if a.Source != nil {
		cpy.Source = a.Source.DeepCopy()
	}
	return &cpy
}

func (a Asset) IntoProto() *pb.Asset {
	return &pb.Asset{
		Id:          a.Id,
		Name:        a.Name,
		Symbol:      a.Symbol,
		TotalSupply: a.TotalSupply.String(),
		Decimals:    uint64(a.Decimals),
		Source:      a.Source.IntoProto(),
	}
}

type is_AssetSource interface {
	is_AssetSource()
}

// AssetSource is an asset source definition
type AssetSource struct {
	// The source
	//
	// Types that are valid to be assigned to Source:
	//	*BuiltinAsset
	//	*ERC20
	Source is_AssetSource
}

func (s AssetSource) DeepCopy() *AssetSource {
	out := s
	switch ss := s.Source.(type) {
	case BuiltinAsset:
		out.Source = ss.DeepCopy()
	case ERC20:
		out.Source = ss.DeepCopy()
	}
	return &out
}

func (a AssetSource) IntoProto() *pb.AssetSource {
	if a.Source == nil {
		return nil
	}
	var out *pb.AssetSource
	switch ss := a.Source.(type) {
	case BuiltinAsset:
		out = &pb.AssetSource{
			Source: &pb.AssetSource_BuiltinAsset{
				BuiltinAsset: &pb.BuiltinAsset{
					Name:                ss.Name,
					Symbol:              ss.Symbol,
					TotalSupply:         ss.TotalSupply.String(),
					Decimals:            ss.Decimals,
					MaxFaucetAmountMint: ss.MaxFaucetAmountMint,
				},
			},
		}
	case ERC20:
		out = &pb.AssetSource{
			Source: &pb.AssetSource_Erc20{
				Erc20: &pb.ERC20{
					ContractAddress: ss.ContractAddress,
				},
			},
		}
	}
	return out
}

// BuiltinAsset is a Vega internal asset.
type BuiltinAsset struct {
	// Name of the asset (e.g: Great British Pound)
	Name string
	// Symbol of the asset (e.g: GBP)
	Symbol string
	// Total circulating supply for the asset
	TotalSupply *num.Uint
	// Number of decimal / precision handled by this asset
	Decimals uint64
	// Maximum amount that can be requested by a party through the built-in asset faucet at a time
	MaxFaucetAmountMint string
}

func (BuiltinAsset) is_AssetSource() {}

func (b BuiltinAsset) DeepCopy() *BuiltinAsset {
	out := b
	out.TotalSupply = b.TotalSupply.Clone()
	return &out
}

// An ERC20 token based asset, living on the ethereum network
type ERC20 struct {
	// The address of the contract for the token, on the ethereum network
	ContractAddress string
}

func (ERC20) is_AssetSource() {}

func (e ERC20) DeepCopy() *ERC20 {
	return &e
}
