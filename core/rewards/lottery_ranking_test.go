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

package rewards

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/require"
)

func TestLotteryRewardScoreSorting(t *testing.T) {
	adjustedRewardScores := []*types.PartyContributionScore{
		{Party: "p1", Score: num.DecimalOne()},
		{Party: "p2", Score: num.DecimalTwo()},
		{Party: "p3", Score: num.DecimalFromFloat(0.01)},
	}
	const layout = "Jan 2, 2006 at 3:04pm"

	timestamp, _ := time.Parse(layout, "Aug 7, 2024 at 12:00pm")
	lottery := lotteryRewardScoreSorting(adjustedRewardScores, timestamp)
	require.Equal(t, "p2", lottery[0].Party)
	require.Equal(t, "p1", lottery[1].Party)
	require.Equal(t, "p3", lottery[2].Party)
}
