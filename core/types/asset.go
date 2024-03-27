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

package types

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

var (
	ErrMissingERC20ContractAddress     = errors.New("missing erc20 contract address")
	ErrMissingBuiltinAssetField        = errors.New("missing builtin asset field")
	ErrInvalidAssetNameEmpty           = errors.New("invalid asset, name must not be empty")
	ErrInvalidAssetSymbolEmpty         = errors.New("invalid asset, symbol must not be empty")
	ErrInvalidAssetQuantumZero         = errors.New("invalid asset, quantum must not be zero")
	ErrLifetimeLimitMustBePositive     = errors.New("lifetime limit must be positive")
	ErrWithdrawThresholdMustBePositive = errors.New("withdraw threshold must be positive")
)

type AssetStatus = vegapb.Asset_Status

const (
	// AssetStatusUnspecified is the default value, always invalid.
	AssetStatusUnspecified AssetStatus = vegapb.Asset_STATUS_UNSPECIFIED
	// AssetStatusProposed states the asset is proposed and under vote.
	AssetStatusProposed AssetStatus = vegapb.Asset_STATUS_PROPOSED
	// AssetStatusRejected states the asset has been rejected from governance.
	AssetStatusRejected AssetStatus = vegapb.Asset_STATUS_REJECTED
	// AssetStatusPendingListing states the asset is pending listing from the bridge.
	AssetStatusPendingListing AssetStatus = vegapb.Asset_STATUS_PENDING_LISTING
	// AssetStatusEnabled states the asset is fully usable in the network.
	AssetStatusEnabled AssetStatus = vegapb.Asset_STATUS_ENABLED
)

type Asset struct {
	// Internal identifier of the asset
	ID string
	// Details of the asset (e.g: Great British Pound)
	Details *AssetDetails
	// Status of the asset
	Status AssetStatus
}

type isAssetDetails interface {
	isAssetDetails()
	adIntoProto() interface{}
	DeepClone() isAssetDetails
	Validate() (ProposalError, error)
	String() string
}

func (a Asset) IntoProto() *vegapb.Asset {
	var details *vegapb.AssetDetails
	if a.Details != nil {
		details = a.Details.IntoProto()
	}
	return &vegapb.Asset{
		Id:      a.ID,
		Details: details,
		Status:  a.Status,
	}
}

func (a Asset) DeepClone() *Asset {
	cpy := a
	if a.Details == nil {
		return &cpy
	}
	cpy.Details.Quantum = a.Details.Quantum
	if a.Details.Source != nil {
		cpy.Details.Source = a.Details.Source.DeepClone()
	}
	return &cpy
}

func AssetFromProto(p *vegapb.Asset) (*Asset, error) {
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
		Status:  p.Status,
	}, nil
}

type AssetDetails struct {
	Name     string
	Symbol   string
	Decimals uint64
	Quantum  num.Decimal
	Source   isAssetDetails
}

func (a AssetDetails) String() string {
	return fmt.Sprintf(
		"name(%s) symbol(%s) quantum(%s) decimals(%d) source(%s)",
		a.Name,
		a.Symbol,
		a.Quantum.String(),
		a.Decimals,
		stringer.ObjToString(a.Source),
	)
}

func (a AssetDetails) Validate() (ProposalError, error) {
	if len(a.Name) == 0 {
		return ProposalErrorInvalidAssetDetails, ErrInvalidAssetNameEmpty
	}

	if len(a.Symbol) == 0 {
		return ProposalErrorInvalidAssetDetails, ErrInvalidAssetSymbolEmpty
	}

	if a.Quantum.IsZero() {
		return ProposalErrorInvalidAssetDetails, ErrInvalidAssetQuantumZero
	}

	return ProposalErrorUnspecified, nil
}

func (a AssetDetails) IntoProto() *vegapb.AssetDetails {
	r := &vegapb.AssetDetails{
		Name:     a.Name,
		Symbol:   a.Symbol,
		Decimals: a.Decimals,
		Quantum:  a.Quantum.String(),
	}
	if a.Source == nil {
		return r
	}
	src := a.Source.adIntoProto()
	switch s := src.(type) {
	case *vegapb.AssetDetails_Erc20:
		r.Source = s
	case *vegapb.AssetDetails_BuiltinAsset:
		r.Source = s
	}
	return r
}

func (a AssetDetails) GetERC20() *ERC20 {
	switch s := a.Source.(type) {
	case *AssetDetailsErc20:
		return s.ERC20
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
	cpy.Quantum = a.Quantum
	return cpy
}

func AssetDetailsFromProto(p *vegapb.AssetDetails) (*AssetDetails, error) {
	var (
		src isAssetDetails
		err error
	)
	switch st := p.Source.(type) {
	case *vegapb.AssetDetails_Erc20:
		src, err = AssetDetailsERC20FromProto(st)
		if err != nil {
			return nil, err
		}
	case *vegapb.AssetDetails_BuiltinAsset:
		src = AssetDetailsBuiltinFromProto(st)
	}
	quantum := num.DecimalZero()
	if len(p.Quantum) > 0 {
		var err error
		quantum, err = num.DecimalFromString(p.Quantum)
		if err != nil {
			return nil, fmt.Errorf("invalid quantum: %w", err)
		}
	}
	return &AssetDetails{
		Name:     p.Name,
		Symbol:   p.Symbol,
		Decimals: p.Decimals,
		Quantum:  quantum,
		Source:   src,
	}, nil
}

type AssetDetailsBuiltinAsset struct {
	BuiltinAsset *BuiltinAsset
}

func (a AssetDetailsBuiltinAsset) String() string {
	return fmt.Sprintf(
		"builtinAsset(%s)",
		stringer.PtrToString(a.BuiltinAsset),
	)
}

func (a AssetDetailsBuiltinAsset) IntoProto() *vegapb.AssetDetails_BuiltinAsset {
	p := &vegapb.AssetDetails_BuiltinAsset{
		BuiltinAsset: &vegapb.BuiltinAsset{},
	}
	if a.BuiltinAsset != nil && a.BuiltinAsset.MaxFaucetAmountMint != nil {
		p.BuiltinAsset.MaxFaucetAmountMint = a.BuiltinAsset.MaxFaucetAmountMint.String()
	}
	return p
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

func (a AssetDetailsBuiltinAsset) Validate() (ProposalError, error) {
	if a.BuiltinAsset.MaxFaucetAmountMint.IsZero() {
		return ProposalErrorMissingBuiltinAssetField, ErrMissingBuiltinAssetField
	}
	return ProposalErrorUnspecified, nil
}

func AssetDetailsBuiltinFromProto(p *vegapb.AssetDetails_BuiltinAsset) *AssetDetailsBuiltinAsset {
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

// BuiltinAsset is a Vega internal asset.
type BuiltinAsset struct {
	MaxFaucetAmountMint *num.Uint
}

func (a BuiltinAsset) String() string {
	return fmt.Sprintf(
		"maxFaucetAmountMint(%s)",
		stringer.PtrToString(a.MaxFaucetAmountMint),
	)
}

type AssetDetailsErc20 struct {
	ERC20 *ERC20
}

func (a AssetDetailsErc20) String() string {
	return fmt.Sprintf(
		"erc20(%s)",
		stringer.PtrToString(a.ERC20),
	)
}

func (a AssetDetailsErc20) IntoProto() *vegapb.AssetDetails_Erc20 {
	lifetimeLimit := "0"
	if a.ERC20.LifetimeLimit != nil {
		lifetimeLimit = a.ERC20.LifetimeLimit.String()
	}
	withdrawThreshold := "0"
	if a.ERC20.WithdrawThreshold != nil {
		withdrawThreshold = a.ERC20.WithdrawThreshold.String()
	}
	return &vegapb.AssetDetails_Erc20{
		Erc20: &vegapb.ERC20{
			ContractAddress:   a.ERC20.ContractAddress,
			ChainId:           a.ERC20.ChainID,
			LifetimeLimit:     lifetimeLimit,
			WithdrawThreshold: withdrawThreshold,
		},
	}
}

func (a AssetDetailsErc20) adIntoProto() interface{} {
	return a.IntoProto()
}

func (AssetDetailsErc20) isAssetDetails() {}

func (a AssetDetailsErc20) DeepClone() isAssetDetails {
	if a.ERC20 == nil {
		return &AssetDetailsErc20{}
	}
	return &AssetDetailsErc20{
		ERC20: a.ERC20.DeepClone(),
	}
}

func (a AssetDetailsErc20) Validate() (ProposalError, error) {
	if len(a.ERC20.ContractAddress) <= 0 {
		return ProposalErrorMissingErc20ContractAddress, ErrMissingERC20ContractAddress
	}

	return ProposalErrorUnspecified, nil
}

func AssetDetailsERC20FromProto(p *vegapb.AssetDetails_Erc20) (*AssetDetailsErc20, error) {
	var (
		lifetimeLimit     = num.UintZero()
		withdrawThreshold = num.UintZero()
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
		ERC20: &ERC20{
			ChainID:           p.Erc20.ChainId,
			ContractAddress:   crypto.EthereumChecksumAddress(p.Erc20.ContractAddress),
			LifetimeLimit:     lifetimeLimit,
			WithdrawThreshold: withdrawThreshold,
		},
	}, nil
}

// An ERC20 token based asset, living on the ethereum network.
type ERC20 struct {
	// Chain ID from which the asset originated from.
	ChainID string

	ContractAddress   string
	LifetimeLimit     *num.Uint
	WithdrawThreshold *num.Uint
}

func (e ERC20) DeepClone() *ERC20 {
	cpy := &ERC20{
		ContractAddress: e.ContractAddress,
		ChainID:         e.ChainID,
	}
	if e.LifetimeLimit != nil {
		cpy.LifetimeLimit = e.LifetimeLimit.Clone()
	} else {
		cpy.LifetimeLimit = num.UintZero()
	}
	if e.WithdrawThreshold != nil {
		cpy.WithdrawThreshold = e.WithdrawThreshold.Clone()
	} else {
		cpy.WithdrawThreshold = num.UintZero()
	}
	return cpy
}

func (e ERC20) String() string {
	return fmt.Sprintf(
		"contractAddress(%s) chainID(%s) lifetimeLimit(%s) withdrawThreshold(%s)",
		e.ContractAddress,
		e.ChainID,
		stringer.PtrToString(e.LifetimeLimit),
		stringer.PtrToString(e.WithdrawThreshold),
	)
}
