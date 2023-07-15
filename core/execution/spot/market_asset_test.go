// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package spot_test

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
