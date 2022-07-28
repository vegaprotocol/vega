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

package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types/num"

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
