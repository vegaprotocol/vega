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
	"time"

	"code.vegaprotocol.io/vega/libs/num"
)

// calculateRewardForProposers calculates the reward given to proposers of markets that crossed the trading threshold for the first time.
func calculateRewardForProposers(epochSeq, asset, accountID string, balance *num.Uint, proposer string, timestamp time.Time) *payout {
	if balance.IsZero() || balance.IsNegative() {
		return nil
	}

	po := &payout{
		asset:         asset,
		fromAccount:   accountID,
		epochSeq:      epochSeq,
		timestamp:     timestamp.Unix(),
		partyToAmount: map[string]*num.Uint{},
	}
	po.partyToAmount[proposer] = balance.Clone()
	po.totalReward = balance.Clone()
	return po
}
