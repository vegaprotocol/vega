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

package stubs

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/common"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type AssetStub struct {
	registered map[string]*assets.Asset
	permissive bool
	defaultDP  uint64
}

func NewAssetStub() *AssetStub {
	return &AssetStub{
		registered: map[string]*assets.Asset{},
		permissive: true,
		defaultDP:  0,
	}
}

func (a *AssetStub) Get(id string) (*assets.Asset, error) {
	r, ok := a.registered[id]
	if ok {
		// pre-registered, so simply return
		return r, nil
	}
	if !a.permissive {
		// we're in strict mode, unknown assets should result in errors
		return nil, errors.New("unknown asset")
	}
	// permissive, we return the default decimal asset
	stub := NewIsAssetStub(id, a.defaultDP, nil)
	return stub, nil
}

func (a *AssetStub) Register(id string, decimals uint64, quantum *num.Decimal) {
	a.registered[id] = NewIsAssetStub(id, decimals, quantum)
}

func (a *AssetStub) SetPermissive() {
	a.permissive = true
}

func (a *AssetStub) SetStrict() {
	a.permissive = false
}

func (AssetStub) Enable(_ context.Context, assetID string) error {
	return nil
}

func (a *AssetStub) ApplyAssetUpdate(_ context.Context, assetID string) error {
	return nil
}

type isAssetStub struct {
	ID            string
	DecimalPlaces uint64
	Status        types.AssetStatus
	Quantum       *num.Decimal
}

func NewIsAssetStub(id string, dp uint64, quantum *num.Decimal) *assets.Asset {
	return assets.NewAsset(&isAssetStub{
		ID:            id,
		DecimalPlaces: dp,
		Status:        types.AssetStatusProposed,
		Quantum:       quantum,
	})
}

func (a isAssetStub) Type() *types.Asset {
	quantum := num.DecimalFromFloat(5000)
	if a.Quantum != nil {
		quantum = *a.Quantum
	}
	return &types.Asset{
		ID: a.ID,
		Details: &types.AssetDetails{
			Decimals: a.DecimalPlaces,
			Quantum:  quantum,
		},
	}
}

func (a *isAssetStub) SetPendingListing() {
	a.Status = types.AssetStatusPendingListing
}

func (a *isAssetStub) SetRejected() {
	a.Status = types.AssetStatusRejected
}

func (a *isAssetStub) SetEnabled() {
	a.Status = types.AssetStatusEnabled
}

func (isAssetStub) GetAssetClass() common.AssetClass {
	return common.Builtin
}

func (isAssetStub) IsValid() bool {
	return true
}

func (isAssetStub) Validate() error {
	return nil
}

func (isAssetStub) SetValid() {}

func (a isAssetStub) String() string {
	return a.ID
}
