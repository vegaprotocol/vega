//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types/num"
)

type Asset struct {
	// Internal identifier of the asset
	Id string
	// Name of the asset (e.g: Great British Pound)
	Details *AssetDetails
}

type AssetDetails struct {
	Name        string
	Symbol      string
	TotalSupply *num.Uint
	Decimals    uint64
	MinLpStake  *num.Uint
	//	*AssetDetails_BuiltinAsset
	//	*AssetDetails_Erc20
	Source isAssetDetails
}

type isAssetDetails interface {
	isAssetDetails()
	adIntoProto() interface{}
	DeepClone() isAssetDetails
}

type AssetDetails_BuiltinAsset struct {
	BuiltinAsset *BuiltinAsset
}

// BuiltinAsset is a Vega internal asset.
type BuiltinAsset struct {
	MaxFaucetAmountMint *num.Uint
}

type AssetDetails_Erc20 struct {
	Erc20 *ERC20
}

// An ERC20 token based asset, living on the ethereum network
type ERC20 struct {
	ContractAddress string
}

func (a Asset) IntoProto() *proto.Asset {
	return &proto.Asset{
		Id:      a.Id,
		Details: a.Details.IntoProto(),
	}
}

func AssetFromProto(p *proto.Asset) *Asset {
	return &Asset{
		Id:      p.Id,
		Details: AssetDetailsFromProto(p.Details),
	}
}

func (a AssetDetails) IntoProto() *proto.AssetDetails {
	src := a.Source.adIntoProto()
	r := &proto.AssetDetails{
		Name:        a.Name,
		Symbol:      a.Symbol,
		TotalSupply: a.TotalSupply.String(),
		Decimals:    a.Decimals,
		MinLpStake:  a.MinLpStake.String(),
	}
	switch s := src.(type) {
	case *proto.AssetDetails_Erc20:
		r.Source = s
	case *proto.AssetDetails_BuiltinAsset:
		r.Source = s
	}
	return r
}

func AssetDetailsFromProto(p *proto.AssetDetails) *AssetDetails {
	var src isAssetDetails
	switch st := p.Source.(type) {
	case *proto.AssetDetails_Erc20:
		src = AssetDetailsERC20FromProto(st)
	case *proto.AssetDetails_BuiltinAsset:
		src = AssetDetailsBuiltinFromProto(st)
	}
	total, _ := num.UintFromString(p.TotalSupply, 10)
	min, _ := num.UintFromString(p.MinLpStake, 10)
	return &AssetDetails{
		Name:        p.Name,
		Symbol:      p.Symbol,
		TotalSupply: total,
		Decimals:    p.Decimals,
		MinLpStake:  min,
		Source:      src,
	}
}

func (a AssetDetails_BuiltinAsset) IntoProto() *proto.AssetDetails_BuiltinAsset {
	return &proto.AssetDetails_BuiltinAsset{
		BuiltinAsset: &proto.BuiltinAsset{
			MaxFaucetAmountMint: a.BuiltinAsset.MaxFaucetAmountMint.String(),
		},
	}
}

func AssetDetailsBuiltinFromProto(p *proto.AssetDetails_BuiltinAsset) *AssetDetails_BuiltinAsset {
	max, _ := num.UintFromString(p.BuiltinAsset.MaxFaucetAmountMint, 10)
	return &AssetDetails_BuiltinAsset{
		BuiltinAsset: &BuiltinAsset{
			MaxFaucetAmountMint: max,
		},
	}
}

func (a AssetDetails_BuiltinAsset) adIntoProto() interface{} {
	return a.IntoProto()
}

func (AssetDetails_BuiltinAsset) isAssetDetails() {}

func (a AssetDetails_BuiltinAsset) DeepClone() isAssetDetails {
	cpy := a
	if a.BuiltinAsset == nil {
		return &cpy
	}
	if a.BuiltinAsset.MaxFaucetAmountMint != nil {
		cpy.BuiltinAsset.MaxFaucetAmountMint = a.BuiltinAsset.MaxFaucetAmountMint.Clone()
	}
	return &cpy
}

func (a AssetDetails_Erc20) IntoProto() *proto.AssetDetails_Erc20 {
	return &proto.AssetDetails_Erc20{
		Erc20: &proto.ERC20{
			ContractAddress: a.Erc20.ContractAddress,
		},
	}
}

func AssetDetailsERC20FromProto(p *proto.AssetDetails_Erc20) *AssetDetails_Erc20 {
	return &AssetDetails_Erc20{
		Erc20: &ERC20{
			ContractAddress: p.Erc20.ContractAddress,
		},
	}
}

func (a AssetDetails_Erc20) adIntoProto() interface{} {
	return a.IntoProto()
}

func (AssetDetails_Erc20) isAssetDetails() {}

func (a AssetDetails_Erc20) DeepClone() isAssetDetails {
	cpy := a
	return &cpy
}

// DeepClone returns a deep clone of a.
func (a Asset) DeepClone() *Asset {
	cpy := a
	if a.Details == nil {
		return &cpy
	}
	if a.Details.TotalSupply != nil {
		cpy.Details.TotalSupply = a.Details.TotalSupply.Clone()
	}
	if a.Details.MinLpStake != nil {
		cpy.Details.MinLpStake = a.Details.MinLpStake.Clone()
	}
	if a.Details.Source != nil {
		cpy.Details.Source = a.Details.Source.DeepClone()
	}
	return &cpy
}

func (a Asset) GetAssetTotalSupply() *num.Uint {
	if a.Details == nil || a.Details.TotalSupply == nil {
		return num.NewUint(0)
	}
	return a.Details.TotalSupply.Clone()
}

func (a AssetDetails) GetErc20() *ERC20 {
	switch s := a.Source.(type) {
	case AssetDetails_Erc20:
		return s.Erc20
	default:
		return nil
	}
}
