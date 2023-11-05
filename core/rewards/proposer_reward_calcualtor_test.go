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

	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/require"
)

func TestProposerBonusCalculator(t *testing.T) {
	now := time.Now()
	require.Nil(t, calculateRewardForProposers("1", "asset", "123456", num.UintZero(), "mememe", now))

	// there's balance in the reward account => the proposer should be paid
	po := calculateRewardForProposers("1", "asset", "123456", num.NewUint(3000), "p1", now)
	require.Equal(t, "asset", po.asset)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, "123456", po.fromAccount)
	require.Equal(t, now.Unix(), po.timestamp)
	require.Equal(t, "3000", po.partyToAmount["p1"].String())
	require.Equal(t, 1, len(po.partyToAmount))
}
