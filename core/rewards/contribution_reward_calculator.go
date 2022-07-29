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
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

// calculateRewardsByContribution calculates the reward based on the fee contribution (whether paid or received) of the parties in the asset.
func calculateRewardsByContribution(epochSeq, asset, accountID string, rewardType types.AccountType, balance *num.Uint, participation []*types.PartyContibutionScore, timestamp time.Time) *payout {
	po := &payout{
		asset:         asset,
		fromAccount:   accountID,
		epochSeq:      epochSeq,
		timestamp:     timestamp.Unix(),
		partyToAmount: map[string]*num.Uint{},
	}
	total := num.UintZero()
	rewardBalance := balance.ToDecimal()
	for _, p := range participation {
		partyReward, _ := num.UintFromDecimal(rewardBalance.Mul(p.Score))
		if !partyReward.IsZero() {
			po.partyToAmount[p.Party] = partyReward
			total.AddSum(partyReward)
		}
	}
	po.totalReward = total
	if total.IsZero() {
		return nil
	}
	return po
}
