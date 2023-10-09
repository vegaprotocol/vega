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

func PartiesShouldHaveTheFollowingStakingAccountBalances(
	stakingAccounts *stubs.StakingAccountStub,
	table *godog.Table,
) error {
	for _, r := range parseStakingAccountBalancesTable(table) {
		row := stakingAccountBalancesRow{row: r}
		balance, err := stakingAccounts.GetAvailableBalance(row.Party())
		if err != nil {
			return err
		}
		if balance.NEQ(row.Amount()) {
			return errMismatchedStakingAccountBalances(row.Party(), row.Amount(), balance)
		}
	}
	return nil
}

func errMismatchedStakingAccountBalances(party string, expectedBalance, actualBalance *num.Uint) error {
	// if bond account was given
	return formatDiff(
		fmt.Sprintf("staking account balances did not match for party(%s)", party),
		map[string]string{
			"staking account balance": u64ToS(expectedBalance.Uint64()),
		},
		map[string]string{
			"staking account balance": u64ToS(actualBalance.Uint64()),
		},
	)
}

func parseStakingAccountBalancesTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"asset",
		"amount",
	}, nil)
}

type stakingAccountBalancesRow struct {
	row RowWrapper
}

func (r stakingAccountBalancesRow) Party() string {
	return r.row.MustStr("party")
}

func (r stakingAccountBalancesRow) Asset() string {
	return r.row.MustStr("asset")
}

func (r stakingAccountBalancesRow) Amount() *num.Uint {
	return num.NewUint(r.row.U64("amount"))
}
