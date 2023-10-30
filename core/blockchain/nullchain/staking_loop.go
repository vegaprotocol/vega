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

package nullchain

import (
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
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
	return num.NewUint(1)
}
