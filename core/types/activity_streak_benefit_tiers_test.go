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

package types_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"
	"github.com/stretchr/testify/require"
)

func TestActivityStreakNetworkParameter(t *testing.T) {
	err := types.CheckActivityStreakBenefitTiers(&proto.ActivityStreakBenefitTiers{})
	require.ErrorIs(t, err, types.ErrNoTierSet)

	// a valid tier
	tier := &proto.ActivityStreakBenefitTier{
		MinimumActivityStreak: 10,
		RewardMultiplier:      "1.01",
		VestingMultiplier:     "1.05",
	}
	tiers := &proto.ActivityStreakBenefitTiers{
		Tiers: []*proto.ActivityStreakBenefitTier{tier},
	}
	err = types.CheckActivityStreakBenefitTiers(tiers)
	require.NoError(t, err)

	// now pick a invalid reward multiplier
	tier.RewardMultiplier = "-100"
	err = types.CheckActivityStreakBenefitTiers(tiers)
	require.Error(t, err)

	// now pick a invalid vesting multiplier
	tier.RewardMultiplier = "1.01"
	tier.VestingMultiplier = "-100"
	err = types.CheckActivityStreakBenefitTiers(tiers)
	require.Error(t, err)

	// now check equals to 1
	tier.RewardMultiplier = "1"
	tier.VestingMultiplier = "1"
	err = types.CheckActivityStreakBenefitTiers(tiers)
	require.NoError(t, err)
}
