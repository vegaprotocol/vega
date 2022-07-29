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

package rewards

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/require"
)

func TestProposerBonusCalculator(t *testing.T) {
	now := time.Now()
	require.Nil(t, calculateRewardForProposers("1", "asset", "123456", types.AccountTypeMarketProposerReward, num.UintZero(), []string{"mememe"}, now))
	require.Nil(t, calculateRewardForProposers("1", "asset", "123456", types.AccountTypeMarketProposerReward, num.NewUint(10000), []string{}, now))

	po := calculateRewardForProposers("1", "asset", "123456", types.AccountTypeMarketProposerReward, num.NewUint(9000), []string{"p1", "p2", "p3"}, now)
	require.Equal(t, "asset", po.asset)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, "123456", po.fromAccount)
	require.Equal(t, now.Unix(), po.timestamp)
	require.Equal(t, "3000", po.partyToAmount["p1"].String())
	require.Equal(t, "3000", po.partyToAmount["p2"].String())
	require.Equal(t, "3000", po.partyToAmount["p3"].String())
	require.Equal(t, 3, len(po.partyToAmount))
}
