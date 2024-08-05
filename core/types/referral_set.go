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
	"time"

	"code.vegaprotocol.io/vega/libs/num"
)

type ReferralSetID string

type ReferralSet struct {
	ID ReferralSetID

	CreatedAt time.Time
	UpdatedAt time.Time

	Referrer *Membership
	Referees []*Membership

	CurrentRewardFactors           Factors
	CurrentRewardsMultiplier       num.Decimal
	CurrentRewardsFactorMultiplier Factors
}

type ReferralSetStats struct {
	AtEpoch                  uint64
	SetID                    ReferralSetID
	WasEligible              bool
	ReferralSetRunningVolume *num.Uint
	ReferrerTakerVolume      *num.Uint
	RefereesStats            map[PartyID]*RefereeStats
	RewardFactors            Factors
	RewardsMultiplier        num.Decimal
	RewardsFactorsMultiplier Factors
}

type RefereeStats struct {
	DiscountFactors Factors
	TakerVolume     *num.Uint
}
