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

package execution_test

import (
	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/common"
	"code.vegaprotocol.io/vega/core/types"
)

type isAssetStub struct {
	ID            string
	DecimalPlaces uint64
	Status        types.AssetStatus
}

func NewAssetStub(id string, dp uint64) *assets.Asset {
	return assets.NewAsset(&isAssetStub{
		ID:            id,
		DecimalPlaces: dp,
		Status:        types.AssetStatusEnabled,
	})
}

func (a isAssetStub) Type() *types.Asset {
	return &types.Asset{
		ID: a.ID,
		Details: &types.AssetDetails{
			Symbol:   a.ID,
			Decimals: a.DecimalPlaces,
		},
		Status: a.Status,
	}
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

func (isAssetStub) SetValid()          {}
func (isAssetStub) SetPendingListing() {}
func (isAssetStub) SetRejected()       {}
func (isAssetStub) SetEnabled()        {}

func (a isAssetStub) String() string {
	return a.ID
}
