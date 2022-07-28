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

package stubs

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/common"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/num"
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
	stub := NewIsAssetStub(id, a.defaultDP)
	return stub, nil
}

func (a *AssetStub) Register(id string, decimals uint64) {
	a.registered[id] = NewIsAssetStub(id, decimals)
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
}

func NewIsAssetStub(id string, dp uint64) *assets.Asset {
	return assets.NewAsset(&isAssetStub{
		ID:            id,
		DecimalPlaces: dp,
		Status:        types.AssetStatusProposed,
	})
}

func (a isAssetStub) Type() *types.Asset {
	return &types.Asset{
		ID: a.ID,
		Details: &types.AssetDetails{
			Decimals: a.DecimalPlaces,
			Quantum:  num.DecimalFromFloat(5000),
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

func (isAssetStub) SetValidNonValidator() {}

func (a isAssetStub) String() string {
	return a.ID
}
