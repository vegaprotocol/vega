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

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

func PartiesShouldHaveTheFollowingDelegationBalances(
	broker *stubs.BrokerStub,
	table *godog.Table,
	epochSeq string,
) error {
	delegationBalances := broker.GetDelegationBalance(epochSeq)

	validatorToAmount := map[string]map[string]string{}
	for _, v := range delegationBalances {
		partyDelegations, ok := validatorToAmount[v.Party]
		if !ok {
			validatorToAmount[v.Party] = map[string]string{}
			partyDelegations = validatorToAmount[v.Party]
		}
		partyDelegations[v.NodeId] = v.Amount
	}

	for _, r := range parseDelegationBalanceTable(table) {
		row := delegationBalanceRow{row: r}

		actualBalance := num.UintZero().String()
		partyDelegations, ok := validatorToAmount[row.Party()]
		if ok {
			if _, ok = partyDelegations[row.NodeID()]; ok {
				actualBalance = partyDelegations[row.NodeID()]
			}
		}

		if row.ExpectedAmount() != actualBalance {
			return errMismatchedBalance(row, actualBalance)
		}
	}
	return nil
}

func errMismatchedBalance(row delegationBalanceRow, selfStake string) error {
	return formatDiff(
		fmt.Sprintf("delegated balances did not match for node(%s)", row.NodeID()),
		map[string]string{
			"delegation balance": row.ExpectedAmount(),
		},
		map[string]string{
			"delegation balance": selfStake,
		},
	)
}

func parseDelegationBalanceTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"node id",
		"amount",
	}, nil)
}

type delegationBalanceRow struct {
	row RowWrapper
}

func (r delegationBalanceRow) Party() string {
	return r.row.MustStr("party")
}

func (r delegationBalanceRow) NodeID() string {
	return r.row.MustStr("node id")
}

func (r delegationBalanceRow) ExpectedAmount() string {
	return r.row.MustStr("amount")
}
