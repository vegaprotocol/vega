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

package assets

import (
	"errors"

	"code.vegaprotocol.io/vega/core/assets/builtin"
	"code.vegaprotocol.io/vega/core/assets/common"
	"code.vegaprotocol.io/vega/core/assets/erc20"
	"code.vegaprotocol.io/vega/core/types"
)

var ErrUpdatingAssetWithDifferentTypeOfAsset = errors.New("updating asset with different type of asset")

type isAsset interface {
	// Type get information about the asset itself
	Type() *types.Asset
	// GetAssetClass get the internal asset class
	GetAssetClass() common.AssetClass
	// IsValid is the order valid / validated with the target chain?
	IsValid() bool
	// Validate is used to check if the assets
	// are present on the target chain
	Validate() error
	// SetValidNonValidator will set an asset as valid
	// without running actual validation, this is used in the
	// context of a non-validator node.
	SetValidNonValidator()
	// SetPendingListing Update the state of the asset to pending for listing
	// on an external bridge
	SetPendingListing()
	// SetRejected Update the state of the asset to rejected
	SetRejected()
	SetEnabled()
	String() string
}

type Asset struct {
	isAsset
}

func NewAsset(a isAsset) *Asset {
	return &Asset{a}
}

func (a *Asset) IsERC20() bool {
	_, ok := a.isAsset.(*erc20.ERC20)
	return ok
}

func (a *Asset) IsBuiltinAsset() bool {
	_, ok := a.isAsset.(*builtin.Builtin)
	return ok
}

func (a *Asset) ERC20() (*erc20.ERC20, bool) {
	asset, ok := a.isAsset.(*erc20.ERC20)
	return asset, ok
}

func (a *Asset) BuiltinAsset() (*builtin.Builtin, bool) {
	asset, ok := a.isAsset.(*builtin.Builtin)
	return asset, ok
}

func (a *Asset) ToAssetType() *types.Asset {
	return a.Type()
}

func (a *Asset) DecimalPlaces() uint64 {
	return a.ToAssetType().Details.Decimals
}

func (a *Asset) Update(updatedAsset *Asset) error {
	if updatedAsset.IsERC20() && a.IsERC20() {
		eth, _ := a.ERC20()
		eth.Update(updatedAsset.Type())
		return nil
	}
	return ErrUpdatingAssetWithDifferentTypeOfAsset
}
