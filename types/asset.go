//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"errors"

	proto "code.vegaprotocol.io/data-node/proto/vega"
	"code.vegaprotocol.io/data-node/types/num"
)

var (
	ErrMissingERC20ContractAddress = errors.New("missing erc20 contract address")
	ErrMissingBuiltinAssetField    = errors.New("missing builtin asset field")
	ErrInvalidAssetDetails         = errors.New("invalid asset details")
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
	//	*AssetDetailsBuiltinAsset
	//	*AssetDetailsErc20
	Source isAssetDetails
}

type isAssetDetails interface {
	isAssetDetails()
	adIntoProto() interface{}
	DeepClone() isAssetDetails
	ValidateAssetSource() (ProposalError, error)
}

type AssetDetailsBuiltinAsset struct {
	BuiltinAsset *BuiltinAsset
}

// BuiltinAsset is a Vega internal asset.
type BuiltinAsset struct {
	MaxFaucetAmountMint *num.Uint
}

type AssetDetailsErc20 struct {
	Erc20 *ERC20
}

// An ERC20 token based asset, living on the ethereum network
type ERC20 struct {
	ContractAddress string
}

func (a Asset) IntoProto() *proto.Asset {
	var details *proto.AssetDetails
	if a.Details != nil {
		details = a.Details.IntoProto()
	}
	return &proto.Asset{
		Id:      a.Id,
		Details: details,
	}
}

func AssetFromProto(p *proto.Asset) *Asset {
	var details *AssetDetails
	if p.Details != nil {
		details = AssetDetailsFromProto(p.Details)
	}
	return &Asset{
		Id:      p.Id,
		Details: details,
	}
}

func (a AssetDetails) String() string {
	return a.IntoProto().String()
}

func (a AssetDetails) IntoProto() *proto.AssetDetails {
	r := &proto.AssetDetails{
		Name:        a.Name,
		Symbol:      a.Symbol,
		TotalSupply: num.UintToString(a.TotalSupply),
		Decimals:    a.Decimals,
		MinLpStake:  num.UintToString(a.MinLpStake),
	}
	if a.Source == nil {
		return r
	}
	src := a.Source.adIntoProto()
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

func (a AssetDetailsBuiltinAsset) IntoProto() *proto.AssetDetails_BuiltinAsset {
	p := &proto.AssetDetails_BuiltinAsset{
		BuiltinAsset: &proto.BuiltinAsset{},
	}
	if a.BuiltinAsset != nil && a.BuiltinAsset.MaxFaucetAmountMint != nil {
		p.BuiltinAsset.MaxFaucetAmountMint = a.BuiltinAsset.MaxFaucetAmountMint.String()
	}
	return p
}

func AssetDetailsBuiltinFromProto(p *proto.AssetDetails_BuiltinAsset) *AssetDetailsBuiltinAsset {
	max, _ := num.UintFromString(p.BuiltinAsset.MaxFaucetAmountMint, 10)
	return &AssetDetailsBuiltinAsset{
		BuiltinAsset: &BuiltinAsset{
			MaxFaucetAmountMint: max,
		},
	}
}

func (a AssetDetailsBuiltinAsset) adIntoProto() interface{} {
	return a.IntoProto()
}

func (AssetDetailsBuiltinAsset) isAssetDetails() {}

func (a AssetDetailsBuiltinAsset) DeepClone() isAssetDetails {
	cpy := a
	if a.BuiltinAsset == nil {
		return &cpy
	}
	if a.BuiltinAsset.MaxFaucetAmountMint != nil {
		cpy.BuiltinAsset.MaxFaucetAmountMint = a.BuiltinAsset.MaxFaucetAmountMint.Clone()
	}
	return &cpy
}

func (a AssetDetailsErc20) IntoProto() *proto.AssetDetails_Erc20 {
	return &proto.AssetDetails_Erc20{
		Erc20: &proto.ERC20{
			ContractAddress: a.Erc20.ContractAddress,
		},
	}
}

func (a AssetDetailsBuiltinAsset) ValidateAssetSource() (ProposalError, error) {
	if a.BuiltinAsset.MaxFaucetAmountMint.IsZero() {
		return ProposalError_PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD, ErrMissingBuiltinAssetField
	}
	return ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

func AssetDetailsERC20FromProto(p *proto.AssetDetails_Erc20) *AssetDetailsErc20 {
	return &AssetDetailsErc20{
		Erc20: &ERC20{
			ContractAddress: p.Erc20.ContractAddress,
		},
	}
}

func (a AssetDetailsErc20) adIntoProto() interface{} {
	return a.IntoProto()
}

func (AssetDetailsErc20) isAssetDetails() {}

func (a AssetDetailsErc20) DeepClone() isAssetDetails {
	if a.Erc20 == nil {
		return &AssetDetailsErc20{}
	}
	return &AssetDetailsErc20{
		Erc20: a.Erc20.DeepClone(),
	}
}

func (a AssetDetailsErc20) ValidateAssetSource() (ProposalError, error) {
	if len(a.Erc20.ContractAddress) <= 0 {
		return ProposalError_PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS, ErrMissingERC20ContractAddress
	}
	return ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
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
		return num.Zero()
	}
	return a.Details.TotalSupply.Clone()
}

func (a AssetDetails) GetErc20() *ERC20 {
	switch s := a.Source.(type) {
	case *AssetDetailsErc20:
		return s.Erc20
	default:
		return nil
	}
}

func (a AssetDetails) DeepClone() *AssetDetails {
	var src isAssetDetails
	if a.Source != nil {
		src = a.Source.DeepClone()
	}
	cpy := &AssetDetails{
		Name:     a.Name,
		Symbol:   a.Symbol,
		Decimals: a.Decimals,
		Source:   src,
	}
	if a.TotalSupply != nil {
		cpy.TotalSupply = a.TotalSupply.Clone()
	} else {
		cpy.TotalSupply = num.Zero()
	}
	if a.MinLpStake != nil {
		cpy.MinLpStake = a.MinLpStake.Clone()
	} else {
		cpy.MinLpStake = num.Zero()
	}
	return cpy
}

func (e ERC20) DeepClone() *ERC20 {
	return &ERC20{
		ContractAddress: e.ContractAddress,
	}
}
