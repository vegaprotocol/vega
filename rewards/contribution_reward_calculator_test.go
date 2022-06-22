// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/require"
)

func TestCalculateRewardsByContribution(t *testing.T) {
	require.Nil(t, calculateRewardsByContribution("1", "ETH", "FROM_ACCOUNT", types.AccountTypeMakerFeeReward, num.Zero(), []*types.PartyContibutionScore{{Party: "party1", Score: num.DecimalFromFloat(0.2)}, {Party: "party2", Score: num.DecimalFromFloat(0.8)}}, time.Now()))
	require.Nil(t, calculateRewardsByContribution("1", "ETH", "FROM_ACCOUNT", types.AccountTypeMakerFeeReward, num.NewUint(1000), []*types.PartyContibutionScore{}, time.Now()))

	po := calculateRewardsByContribution("1", "ETH", "FROM_ACCOUNT", types.AccountTypeMakerFeeReward, num.NewUint(1000), []*types.PartyContibutionScore{{Party: "party1", Score: num.DecimalFromFloat(0.2)}, {Party: "party2", Score: num.DecimalFromFloat(0.8)}}, time.Now())
	require.Equal(t, "ETH", po.asset)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, "FROM_ACCOUNT", po.fromAccount)
	require.Equal(t, "200", po.partyToAmount["party1"].String())
	require.Equal(t, "800", po.partyToAmount["party2"].String())
	require.Equal(t, 2, len(po.partyToAmount))
}
