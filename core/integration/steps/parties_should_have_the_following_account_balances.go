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

	"github.com/cucumber/godog"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/integration/stubs"
)

func PartiesShouldHaveTheFollowingAccountBalances(
	broker *stubs.BrokerStub,
	table *godog.Table,
) error {
	for _, r := range parseAccountBalancesTable(table) {
		row := accountBalancesRow{row: r}
		var hasError bool

		generalAccount, err := broker.GetPartyGeneralAccount(row.Party(), row.Asset())
		if err != nil {
			return errCannotGetPartyGeneralAccount(row.Party(), row.Asset(), err)
		}
		if generalAccount.GetBalance() != row.GeneralAccountBalance() {
			hasError = true
		}

		marginAccount, err := broker.GetPartyMarginAccount(row.Party(), row.MarketID())
		if err != nil {
			return errCannotGetPartyMarginAccount(row.Party(), row.Asset(), err)
		}
		// check bond
		var bondAcc types.Account
		if row.ExpectBondAccountBalance() {
			bondAcc, err = broker.GetPartyBondAccountForMarket(row.Party(), row.Asset(), row.MarketID())
			if err == nil && bondAcc.Balance != row.BondAccountBalance() {
				hasError = true
			}
		}
		if marginAccount.GetBalance() != row.MarginAccountBalance() {
			hasError = true
		}

		if hasError {
			return errMismatchedAccountBalances(row, marginAccount, generalAccount, bondAcc)
		}
	}
	return nil
}

func errCannotGetPartyGeneralAccount(party, asset string, err error) error {
	return fmt.Errorf("couldn't get general account for party(%s) and asset(%s): %s",
		party, asset, err.Error(),
	)
}

func errCannotGetPartyMarginAccount(party, asset string, err error) error {
	return fmt.Errorf("couldn't get margin account for party(%s) and asset(%s): %s",
		party, asset, err.Error(),
	)
}

func errMismatchedAccountBalances(row accountBalancesRow, marginAccount, generalAccount, bondAcc types.Account) error {
	// if bond account was given
	if bondAcc.Type == types.AccountType_ACCOUNT_TYPE_BOND {
		return formatDiff(
			fmt.Sprintf("account balances did not match for party(%s)", row.Party()),
			map[string]string{
				"margin account balance":  row.MarginAccountBalance(),
				"general account balance": row.GeneralAccountBalance(),
				"bond account balance":    row.BondAccountBalance(),
			},
			map[string]string{
				"margin account balance":  marginAccount.GetBalance(),
				"general account balance": generalAccount.GetBalance(),
				"bond account balance":    bondAcc.Balance,
			},
		)
	}
	return formatDiff(
		fmt.Sprintf("account balances did not match for party(%s)", row.Party()),
		map[string]string{
			"margin account balance":  row.MarginAccountBalance(),
			"general account balance": row.GeneralAccountBalance(),
		},
		map[string]string{
			"margin account balance":  marginAccount.GetBalance(),
			"general account balance": generalAccount.GetBalance(),
		},
	)
}

func parseAccountBalancesTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"asset",
		"market id",
		"margin",
		"general",
	}, []string{
		"bond",
	})
}

type accountBalancesRow struct {
	row RowWrapper
}

func (r accountBalancesRow) Party() string {
	return r.row.MustStr("party")
}

func (r accountBalancesRow) Asset() string {
	return r.row.MustStr("asset")
}

func (r accountBalancesRow) MarketID() string {
	return r.row.MustStr("market id")
}

func (r accountBalancesRow) MarginAccountBalance() string {
	return r.row.MustStr("margin")
}

func (r accountBalancesRow) GeneralAccountBalance() string {
	return r.row.MustStr("general")
}

func (r accountBalancesRow) ExpectBondAccountBalance() bool {
	return r.row.HasColumn("bond")
}

func (r accountBalancesRow) BondAccountBalance() string {
	return r.row.MustStr("bond")
}
