package types

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

var (
	ErrInvalidLifetimeLimit     = errors.New("invalid lifetime limit")
	ErrInvalidWithdrawThreshold = errors.New("invalid withdraw threshold")
	ErrAssetIDIsRequired        = errors.New("asset ID is required")
	ErrChangesAreRequired       = errors.New("changes are required")
	ErrSourceIsRequired         = errors.New("source is required")
)

type ProposalTermsUpdateAsset struct {
	UpdateAsset *UpdateAsset
}

func (a ProposalTermsUpdateAsset) String() string {
	return fmt.Sprintf(
		"updateAsset(%v)",
		stringer.ReflectPointerToString(a.UpdateAsset),
	)
}

func (a ProposalTermsUpdateAsset) IntoProto() *vegapb.ProposalTerms_UpdateAsset {
	var updateAsset *vegapb.UpdateAsset
	if a.UpdateAsset != nil {
		updateAsset = a.UpdateAsset.IntoProto()
	}
	return &vegapb.ProposalTerms_UpdateAsset{
		UpdateAsset: updateAsset,
	}
}

func (a ProposalTermsUpdateAsset) isPTerm() {}

func (a ProposalTermsUpdateAsset) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsUpdateAsset) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateAsset
}

func (a ProposalTermsUpdateAsset) DeepClone() proposalTerm {
	if a.UpdateAsset == nil {
		return &ProposalTermsUpdateAsset{}
	}
	return &ProposalTermsUpdateAsset{
		UpdateAsset: a.UpdateAsset.DeepClone(),
	}
}

func NewUpdateAssetFromProto(p *vegapb.ProposalTerms_UpdateAsset) (*ProposalTermsUpdateAsset, error) {
	var updateAsset *UpdateAsset
	if p.UpdateAsset != nil {
		updateAsset = &UpdateAsset{
			AssetID: p.UpdateAsset.GetAssetId(),
		}

		if p.UpdateAsset.Changes != nil {
			var err error
			updateAsset.Changes, err = AssetDetailsUpdateFromProto(p.UpdateAsset.Changes)
			if err != nil {
				return nil, err
			}
		}
	}

	return &ProposalTermsUpdateAsset{
		UpdateAsset: updateAsset,
	}, nil
}

type UpdateAsset struct {
	AssetID string
	Changes *AssetDetailsUpdate
}

func (a *UpdateAsset) GetChanges() *AssetDetailsUpdate {
	if a != nil {
		return a.Changes
	}
	return nil
}

func (a UpdateAsset) IntoProto() *vegapb.UpdateAsset {
	var changes *vegapb.AssetDetailsUpdate
	if a.Changes != nil {
		changes = a.Changes.IntoProto()
	}
	return &vegapb.UpdateAsset{
		AssetId: a.AssetID,
		Changes: changes,
	}
}

func (a UpdateAsset) String() string {
	return fmt.Sprintf(
		"assetID(%s) changes(%s)",
		a.AssetID,
		stringer.ReflectPointerToString(a.Changes),
	)
}

func (a UpdateAsset) DeepClone() *UpdateAsset {
	cpy := &UpdateAsset{
		AssetID: a.AssetID,
	}

	if a.Changes != nil {
		cpy.Changes = a.Changes.DeepClone()
	}

	return cpy
}

func (a UpdateAsset) Validate() (ProposalError, error) {
	if len(a.AssetID) == 0 {
		return ProposalErrorInvalidAsset, ErrAssetIDIsRequired
	}

	if a.Changes == nil {
		return ProposalErrorInvalidAsset, ErrChangesAreRequired
	}

	return a.Changes.Validate()
}

type AssetDetailsUpdate struct {
	Quantum num.Decimal
	//	*AssetDetailsUpdateERC20
	Source isAssetDetailsUpdate
}

func (a AssetDetailsUpdate) String() string {
	return fmt.Sprintf(
		"quantum(%s) (%s)",
		a.Quantum.String(),
		stringer.ReflectPointerToString(a.Source),
	)
}

func (a AssetDetailsUpdate) IntoProto() *vegapb.AssetDetailsUpdate {
	r := &vegapb.AssetDetailsUpdate{
		Quantum: a.Quantum.String(),
	}
	if a.Source == nil {
		return r
	}
	src := a.Source.aduIntoProto()
	switch s := src.(type) {
	case *vegapb.AssetDetailsUpdate_Erc20:
		r.Source = s
	}
	return r
}

func (a AssetDetailsUpdate) DeepClone() *AssetDetailsUpdate {
	var src isAssetDetailsUpdate
	if a.Source != nil {
		src = a.Source.DeepClone()
	}
	cpy := &AssetDetailsUpdate{
		Source: src,
	}
	cpy.Quantum = a.Quantum
	return cpy
}

func (a AssetDetailsUpdate) Validate() (ProposalError, error) {
	if a.Quantum.IsZero() {
		return ProposalErrorInvalidAssetDetails, ErrInvalidAssetQuantumZero
	}

	if a.Source == nil {
		return ProposalErrorInvalidAssetDetails, ErrSourceIsRequired
	}

	return a.Source.Validate()
}

func AssetDetailsUpdateFromProto(p *vegapb.AssetDetailsUpdate) (*AssetDetailsUpdate, error) {
	var (
		src isAssetDetailsUpdate
		err error
	)
	switch st := p.Source.(type) {
	case *vegapb.AssetDetailsUpdate_Erc20:
		src, err = AssetDetailsUpdateERC20FromProto(st)
		if err != nil {
			return nil, err
		}
	}

	min := num.DecimalZero()
	if len(p.Quantum) > 0 {
		var err error
		min, err = num.DecimalFromString(p.Quantum)
		if err != nil {
			return nil, fmt.Errorf("invalid quantum: %w", err)
		}
	}

	return &AssetDetailsUpdate{
		Quantum: min,
		Source:  src,
	}, nil
}

type isAssetDetailsUpdate interface {
	isAssetDetailsUpdate()
	aduIntoProto() interface{}
	DeepClone() isAssetDetailsUpdate
	Validate() (ProposalError, error)
	String() string
}

type AssetDetailsUpdateERC20 struct {
	ERC20Update *ERC20Update
}

func (a AssetDetailsUpdateERC20) String() string {
	return fmt.Sprintf(
		"erc20Update(%s)",
		stringer.ReflectPointerToString(a.ERC20Update),
	)
}

func (a AssetDetailsUpdateERC20) aduIntoProto() interface{} {
	return a.IntoProto()
}

func (AssetDetailsUpdateERC20) isAssetDetailsUpdate() {}

func (a AssetDetailsUpdateERC20) DeepClone() isAssetDetailsUpdate {
	if a.ERC20Update == nil {
		return &AssetDetailsUpdateERC20{}
	}
	return &AssetDetailsUpdateERC20{
		ERC20Update: a.ERC20Update.DeepClone(),
	}
}

func (a AssetDetailsUpdateERC20) IntoProto() *vegapb.AssetDetailsUpdate_Erc20 {
	lifetimeLimit := "0"
	if a.ERC20Update.LifetimeLimit != nil {
		lifetimeLimit = a.ERC20Update.LifetimeLimit.String()
	}
	withdrawThreshold := "0"
	if a.ERC20Update.WithdrawThreshold != nil {
		withdrawThreshold = a.ERC20Update.WithdrawThreshold.String()
	}
	return &vegapb.AssetDetailsUpdate_Erc20{
		Erc20: &vegapb.ERC20Update{
			LifetimeLimit:     lifetimeLimit,
			WithdrawThreshold: withdrawThreshold,
		},
	}
}

func (a AssetDetailsUpdateERC20) Validate() (ProposalError, error) {
	if a.ERC20Update.LifetimeLimit.EQ(num.UintZero()) {
		return ProposalErrorInvalidAsset, ErrLifetimeLimitMustBePositive
	}

	if a.ERC20Update.WithdrawThreshold.EQ(num.UintZero()) {
		return ProposalErrorInvalidAsset, ErrWithdrawThresholdMustBePositive
	}

	return ProposalErrorUnspecified, nil
}

func AssetDetailsUpdateERC20FromProto(p *vegapb.AssetDetailsUpdate_Erc20) (*AssetDetailsUpdateERC20, error) {
	var (
		lifetimeLimit     = num.UintZero()
		withdrawThreshold = num.UintZero()
		overflow          bool
	)
	if len(p.Erc20.LifetimeLimit) > 0 {
		lifetimeLimit, overflow = num.UintFromString(p.Erc20.LifetimeLimit, 10)
		if overflow {
			return nil, ErrInvalidLifetimeLimit
		}
		if lifetimeLimit.EQ(num.UintZero()) {
			return nil, ErrLifetimeLimitMustBePositive
		}
	}
	if len(p.Erc20.WithdrawThreshold) > 0 {
		withdrawThreshold, overflow = num.UintFromString(p.Erc20.WithdrawThreshold, 10)
		if overflow {
			return nil, ErrInvalidWithdrawThreshold
		}
		if withdrawThreshold.EQ(num.UintZero()) {
			return nil, ErrWithdrawThresholdMustBePositive
		}
	}
	return &AssetDetailsUpdateERC20{
		ERC20Update: &ERC20Update{
			LifetimeLimit:     lifetimeLimit,
			WithdrawThreshold: withdrawThreshold,
		},
	}, nil
}

type ERC20Update struct {
	LifetimeLimit     *num.Uint
	WithdrawThreshold *num.Uint
}

func (e ERC20Update) DeepClone() *ERC20Update {
	cpy := &ERC20Update{}

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

func (e ERC20Update) String() string {
	return fmt.Sprintf(
		"lifetimeLimit(%s) withdrawThreshold(%s)",
		stringer.UintPointerToString(e.LifetimeLimit),
		stringer.UintPointerToString(e.WithdrawThreshold),
	)
}
