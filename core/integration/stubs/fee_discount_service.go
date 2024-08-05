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
	"errors"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type ReferralDiscountRewardService struct{}

func (*ReferralDiscountRewardService) ReferralDiscountFactorsForParty(party types.PartyID) num.Decimal {
	return num.DecimalZero()
}

func (*ReferralDiscountRewardService) RewardsFactorsMultiplierAppliedForParty(party types.PartyID) num.Decimal {
	return num.DecimalZero()
}

func (*ReferralDiscountRewardService) GetReferrer(referee types.PartyID) (types.PartyID, error) {
	return types.PartyID(""), errors.New("no referrer")
}

type VolumeDiscountService struct{}

func (*VolumeDiscountService) VolumeDiscountFactorForParty(party types.PartyID) num.Decimal {
	return num.DecimalZero()
}
