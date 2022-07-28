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

package nullchain

import (
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/num"
)

type Collateral interface {
	GetPartyGeneralAccount(party, asset string) (*types.Account, error)
}

type Assets interface {
	Get(assetID string) (*assets.Asset, error)
}

type StakingLoop struct {
	col    Collateral
	assets Assets

	// The built-in asset which when deposited into the collateral is to be used to pretend that is was
	// staked on the bridge
	stakingAsset string
}

// NewStakingLoop return a type that can "mock" a StakingAccount by instead reading deposited amounts
// from the collateral engine. Used by the null-blockchain to remove the need for an Ethereum connection.
func NewStakingLoop(col Collateral, assets Assets) *StakingLoop {
	return &StakingLoop{
		col:          col,
		assets:       assets,
		stakingAsset: "VOTE",
	}
}

func (s *StakingLoop) GetAvailableBalance(party string) (*num.Uint, error) {
	acc, err := s.col.GetPartyGeneralAccount(party, s.stakingAsset)
	if err != nil {
		return nil, err
	}
	return acc.Balance.Clone(), nil
}

func (s *StakingLoop) GetAvailableBalanceInRange(party string, from, to time.Time) (*num.Uint, error) {
	// We're just going to have to say we have no notion of time range and whatever is has be deposited by the faucet
	// has always been there.
	return s.GetAvailableBalance(party)
}

func (s *StakingLoop) GetStakingAssetTotalSupply() *num.Uint {
	asset, err := s.assets.Get(s.stakingAsset)
	if err != nil {
		return nil
	}
	return asset.Type().GetAssetTotalSupply()
}
