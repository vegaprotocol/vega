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
	// SetPendingListing Update the state of the asset to pending for listing
	// on an external bridge
	SetPendingListing()
	// SetRejected Update the state of the asset to rejected
	SetRejected()
	SetEnabled()
	SetValid()
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
