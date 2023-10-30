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

package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/integration/stubs"
)

func PartiesShouldReceiveTheFollowingReward(
	broker *stubs.BrokerStub,
	table *godog.Table,
	epochSeq string,
) error {
	rewards := broker.GetRewards(epochSeq)

	for _, r := range parseRewardsTable(table) {
		row := rewardRow{row: r}

		actualReward := num.UintZero().String()
		if reward, ok := rewards[stubs.AssetParty{Asset: row.Asset(), Party: row.Party()}]; ok {
			actualReward = reward.Amount.String()
		}

		if row.Amount() != actualReward {
			return errMismatchedReward(row, actualReward)
		}
	}
	return nil
}

func errMismatchedReward(row rewardRow, actualReward string) error {
	return formatDiff(
		fmt.Sprintf("reward amount did not match for party(%s)", row.Party()),
		map[string]string{
			"reward amount": row.Amount(),
		},
		map[string]string{
			"reward amount": actualReward,
		},
	)
}

func parseRewardsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"asset",
		"amount",
	}, nil)
}

type rewardRow struct {
	row RowWrapper
}

func (r rewardRow) Asset() string {
	return r.row.MustStr("asset")
}

func (r rewardRow) Party() string {
	return r.row.MustStr("party")
}

func (r rewardRow) Amount() string {
	return r.row.MustStr("amount")
}
