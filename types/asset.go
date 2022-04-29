package types

import (
	"errors"
	"fmt"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
)

var (
	ErrMissingERC20ContractAddress = errors.New("missing erc20 contract address")
	ErrMissingBuiltinAssetField    = errors.New("missing builtin asset field")
	ErrInvalidAssetDetails         = errors.New("invalid asset details")

	ErrInvalidAssetNameEmpty         = errors.New("invalid asset, name must not be empty")
	ErrInvalidAssetSymbolEmpty       = errors.New("invalid asset, symbol must not be empty")
	ErrInvalidAssetDecimalPlacesZero = errors.New("invalid asset, decimal places must not be zero")
	ErrInvalidAssetTotalSupplyZero   = errors.New("invalid asset, total supply must not be zero")
	ErrInvalidAssetQuantumZero       = errors.New("invalid asset, quantum must not be zero")
)

type Asset struct {
	// Internal identifier of the asset
	ID string
	// Name of the asset (e.g: Great British Pound)
	Details *AssetDetails
}

type AssetDetails struct {
	Name        string
	Symbol      string
	TotalSupply *num.Uint
	Decimals    uint64
	Quantum     num.Decimal
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

// An ERC20 token based asset, living on the ethereum network.
type ERC20 struct {
	ContractAddress   string
	LifetimeLimit     *num.Uint
	WithdrawThreshold *num.Uint
}

func (a Asset) IntoProto() *proto.Asset {
	var details *proto.AssetDetails
	if a.Details != nil {
		details = a.Details.IntoProto()
	}
	return &proto.Asset{
		Id:      a.ID,
		Details: details,
	}
}

func AssetFromProto(p *proto.Asset) (*Asset, error) {
	var (
		details *AssetDetails
		err     error
	)
	if p.Details != nil {
		details, err = AssetDetailsFromProto(p.Details)
		if err != nil {
			return nil, err
		}
	}
	return &Asset{
		ID:      p.Id,
		Details: details,
	}, nil
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
		Quantum:     a.Quantum.String(),
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

func AssetDetailsFromProto(p *proto.AssetDetails) (*AssetDetails, error) {
	var (
		src isAssetDetails
		err error
	)
	switch st := p.Source.(type) {
	case *proto.AssetDetails_Erc20:
		src, err = AssetDetailsERC20FromProto(st)
		if err != nil {
			return nil, err
		}
	case *proto.AssetDetails_BuiltinAsset:
		src = AssetDetailsBuiltinFromProto(st)
	}
	total := num.Zero()
	min := num.DecimalZero()
	if len(p.TotalSupply) > 0 {
		var overflow bool
		total, overflow = num.UintFromString(p.TotalSupply, 10)
		if overflow {
			return nil, errors.New("invalid total supply")
		}
	}
	if len(p.Quantum) > 0 {
		var err error
		min, err = num.DecimalFromString(p.Quantum)
		if err != nil {
			return nil, fmt.Errorf("invalid quantum: %w", err)
		}
	}
	return &AssetDetails{
		Name:        p.Name,
		Symbol:      p.Symbol,
		TotalSupply: total,
		Decimals:    p.Decimals,
		Quantum:     min,
		Source:      src,
	}, nil
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
	if p.BuiltinAsset.MaxFaucetAmountMint != "" {
		max, _ := num.UintFromString(p.BuiltinAsset.MaxFaucetAmountMint, 10)
		return &AssetDetailsBuiltinAsset{
			BuiltinAsset: &BuiltinAsset{
				MaxFaucetAmountMint: max,
			},
		}
	}
	return &AssetDetailsBuiltinAsset{}
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
	lifetimeLimit := "0"
	if a.Erc20.LifetimeLimit != nil {
		lifetimeLimit = a.Erc20.LifetimeLimit.String()
	}
	withdrawThreshold := "0"
	if a.Erc20.WithdrawThreshold != nil {
		withdrawThreshold = a.Erc20.WithdrawThreshold.String()
	}
	return &proto.AssetDetails_Erc20{
		Erc20: &proto.ERC20{
			ContractAddress:   a.Erc20.ContractAddress,
			LifetimeLimit:     lifetimeLimit,
			WithdrawThreshold: withdrawThreshold,
		},
	}
}

func (a AssetDetailsBuiltinAsset) ValidateAssetSource() (ProposalError, error) {
	if a.BuiltinAsset.MaxFaucetAmountMint.IsZero() {
		return ProposalErrorMissingBuiltinAssetField, ErrMissingBuiltinAssetField
	}
	return ProposalErrorUnspecified, nil
}

func AssetDetailsERC20FromProto(p *proto.AssetDetails_Erc20) (*AssetDetailsErc20, error) {
	var (
		lifetimeLimit     = num.Zero()
		withdrawThreshold = num.Zero()
		overflow          bool
	)
	if len(p.Erc20.LifetimeLimit) > 0 {
		lifetimeLimit, overflow = num.UintFromString(p.Erc20.LifetimeLimit, 10)
		if overflow {
			return nil, errors.New("invalid lifetime limit")
		}
	}
	if len(p.Erc20.WithdrawThreshold) > 0 {
		withdrawThreshold, overflow = num.UintFromString(p.Erc20.WithdrawThreshold, 10)
		if overflow {
			return nil, errors.New("invalid withdraw threshold")
		}
	}
	return &AssetDetailsErc20{
		Erc20: &ERC20{
			ContractAddress:   p.Erc20.ContractAddress,
			LifetimeLimit:     lifetimeLimit,
			WithdrawThreshold: withdrawThreshold,
		},
	}, nil
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
		return ProposalErrorMissingErc20ContractAddress, ErrMissingERC20ContractAddress
	}
	return ProposalErrorUnspecified, nil
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
	cpy.Details.Quantum = a.Details.Quantum
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
	cpy.Quantum = a.Quantum
	return cpy
}

func (e ERC20) DeepClone() *ERC20 {
	cpy := &ERC20{
		ContractAddress: e.ContractAddress,
	}
	if e.LifetimeLimit != nil {
		cpy.LifetimeLimit = e.LifetimeLimit.Clone()
	} else {
		cpy.LifetimeLimit = num.Zero()
	}
	if e.WithdrawThreshold != nil {
		cpy.WithdrawThreshold = e.WithdrawThreshold.Clone()
	} else {
		cpy.WithdrawThreshold = num.Zero()
	}
	return cpy
}
