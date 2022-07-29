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

// calculateRewardForProposers calculates the reward given to proposers of markets that crossed the trading threshold for the first time.
func calculateRewardForProposers(epochSeq, asset, accountID string, rewardType types.AccountType, balance *num.Uint, proposers []string, timestamp time.Time) *payout {
	if len(proposers) <= 0 || balance.IsZero() || balance.IsNegative() {
		return nil
	}

	po := &payout{
		asset:         asset,
		fromAccount:   accountID,
		epochSeq:      epochSeq,
		timestamp:     timestamp.Unix(),
		partyToAmount: map[string]*num.Uint{},
	}
	total := num.UintZero()
	rewardBalance := balance.ToDecimal()
	proposerBonus := rewardBalance.Div(num.DecimalFromInt64(int64(len(proposers))))
	for _, p := range proposers {
		partyReward, _ := num.UintFromDecimal(proposerBonus)
		if !partyReward.IsZero() {
			po.partyToAmount[p] = partyReward
			total.AddSum(partyReward)
		}
	}
	po.totalReward = total
	if total.IsZero() {
		return nil
	}
	return po
}
