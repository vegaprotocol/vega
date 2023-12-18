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

	"github.com/cucumber/godog"
)

func PartiesShouldHaveTheFollowingAccountBalances(
	broker *stubs.BrokerStub,
	table *godog.Table,
) error {
	for _, r := range parseAccountBalancesTable(table) {
		row := accountBalancesRow{row: r}

		expectedValues := map[string]string{}
		foundValues := map[string]string{}

		expectedAsset := row.Asset()
		if row.ExpectGeneralAccountBalance() && len(row.GeneralAccountBalance()) > 0 {
			generalAccount, err := broker.GetPartyGeneralAccount(row.Party(), expectedAsset)
			if err != nil {
				return errCannotGetPartyGeneralAccount(row.Party(), expectedAsset, err)
			}
			if generalAccount.GetAsset() != expectedAsset {
				return errWrongGeneralAccountAsset(row.Party(), expectedAsset, generalAccount.GetAsset())
			}

			foundBalance := generalAccount.GetBalance()
			expectedBalance := row.GeneralAccountBalance()
			if foundBalance != expectedBalance {
				expectedValues["general"] = expectedBalance
				foundValues["general"] = foundBalance
			}
		}

		if row.ExpectMarginAccountBalance() && len(row.MarginAccountBalance()) > 0 {
			if !row.ExpectMarketID() {
				return fmt.Errorf("market id must be specified when expected margin account balance is supplied")
			}
			marginAccount, err := broker.GetPartyMarginAccount(row.Party(), row.MarketID())
			if err != nil {
				return errCannotGetPartyMarginAccount(row.Party(), row.MarketID(), err)
			}
			if marginAccount.GetAsset() != expectedAsset {
				return errWrongMarketAccountAsset(marginAccount.GetType().String(), row.Party(), row.MarketID(), expectedAsset, marginAccount.GetAsset())
			}
			foundBalance := marginAccount.GetBalance()
			expectedBalance := row.MarginAccountBalance()
			if foundBalance != expectedBalance {
				expectedValues["margin"] = expectedBalance
				foundValues["margin"] = foundBalance
			}
		}

		if row.ExpectOrderMarginAccountBalance() && len(row.OrderMarginAccountBalance()) > 0 {
			if !row.ExpectMarketID() {
				return fmt.Errorf("market id must be specified when expected order margin account balance is supplied")
			}
			orderMarginAccount, err := broker.GetPartyOrderMarginAccount(row.Party(), row.MarketID())
			if err != nil {
				return errCannotGetPartyOrderMarginAccount(row.Party(), row.MarketID(), err)
			}
			if orderMarginAccount.GetAsset() != expectedAsset {
				return errWrongMarketAccountAsset(orderMarginAccount.GetType().String(), row.Party(), row.MarketID(), expectedAsset, orderMarginAccount.GetAsset())
			}
			foundBalance := orderMarginAccount.GetBalance()
			expectedBalance := row.OrderMarginAccountBalance()
			if foundBalance != expectedBalance {
				expectedValues["order margin"] = expectedBalance
				foundValues["order margin"] = foundBalance
			}
		}

		// check bond
		if row.ExpectBondAccountBalance() && len(row.BondAccountBalance()) > 0 {
			if !row.ExpectMarketID() {
				return fmt.Errorf("market id must be specified when expected bond account balance is supplied")
			}
			bondAcc, err := broker.GetPartyBondAccountForMarket(row.Party(), expectedAsset, row.MarketID())
			if err != nil {
				return errCannotGetPartyBondAccount(row.Party(), row.MarketID(), err)
			}
			if bondAcc.GetAsset() != expectedAsset {
				return errWrongMarketAccountAsset(bondAcc.GetType().String(), row.Party(), row.MarketID(), expectedAsset, bondAcc.GetAsset())
			}
			foundBalance := bondAcc.GetBalance()
			expectedBalance := row.BondAccountBalance()
			if foundBalance != expectedBalance {
				expectedValues["bond"] = expectedBalance
				foundValues["bond"] = foundBalance
			}
		}

		if row.ExpectVestingAccountBalance() && len(row.VestingAccountBalance()) > 0 {
			if !row.ExpectMarketID() {
				return fmt.Errorf("market id must be specified when expected bond account balance is supplied")
			}
			vestingAcc, err := broker.GetPartyVestingAccountForMarket(row.Party(), expectedAsset, row.MarketID())
			if err != nil {
				return errCannotGetPartyVestingAccount(row.Party(), row.MarketID(), err)
			}
			if vestingAcc.GetAsset() != expectedAsset {
				return errWrongMarketAccountAsset(vestingAcc.GetType().String(), row.Party(), row.MarketID(), expectedAsset, vestingAcc.GetAsset())
			}
			foundBalance := vestingAcc.GetBalance()
			expectedBalance := row.VestingAccountBalance()
			if foundBalance != expectedBalance {
				expectedValues["vesting"] = expectedBalance
				foundValues["vesting"] = foundBalance
			}
		}

		if row.ExpectVestedAccountBalance() && len(row.VestedAccountBalance()) > 0 {
			if !row.ExpectMarketID() {
				return fmt.Errorf("market id must be specified when expected bond account balance is supplied")
			}
			vestedAcc, err := broker.GetPartyVestedAccountForMarket(row.Party(), expectedAsset, row.MarketID())
			if err != nil {
				return errCannotGetPartyVestedAccount(row.Party(), row.MarketID(), err)
			}
			if vestedAcc.GetAsset() != expectedAsset {
				return errWrongMarketAccountAsset(vestedAcc.GetType().String(), row.Party(), row.MarketID(), expectedAsset, vestedAcc.GetAsset())
			}
			foundBalance := vestedAcc.GetBalance()
			expectedBalance := row.VestedAccountBalance()
			if foundBalance != expectedBalance {
				expectedValues["vested"] = expectedBalance
				foundValues["vested"] = foundBalance
			}
		}

		if len(expectedValues) > 0 {
			return formatDiff(fmt.Sprintf("account balances did not match for party %q", row.Party()), expectedValues, foundValues)
		}
	}
	return nil
}

func errCannotGetPartyGeneralAccount(party, asset string, err error) error {
	return fmt.Errorf("couldn't get general account for party(%s) and asset(%s): %w",
		party, asset, err,
	)
}

func errCannotGetPartyOrderMarginAccount(party, market string, err error) error {
	return fmt.Errorf("couldn't get order margin account for party(%s) and market(%s): %w",
		party, market, err,
	)
}

func errCannotGetPartyMarginAccount(party, market string, err error) error {
	return fmt.Errorf("couldn't get margin account for party(%s) and market(%s): %w",
		party, market, err,
	)
}

func errCannotGetPartyBondAccount(party, market string, err error) error {
	return fmt.Errorf("couldn't get bond account for party(%s) and market(%s): %w",
		party, market, err,
	)
}

func errCannotGetPartyVestingAccount(party, market string, err error) error {
	return fmt.Errorf("couldn't get vesting account for party(%s) and market(%s): %w",
		party, market, err,
	)
}

func errCannotGetPartyVestedAccount(party, market string, err error) error {
	return fmt.Errorf("couldn't get vested account for party(%s) and market(%s): %w",
		party, market, err,
	)
}

func errWrongMarketAccountAsset(account, party, market, expectedAsset, actualAsset string) error {
	return fmt.Errorf("%s account for party(%s) in market(%s) uses '%s' asset, but '%s' was expected",
		account, party, market, actualAsset, expectedAsset,
	)
}

func errWrongGeneralAccountAsset(party, expectedAsset, actualAsset string) error {
	return fmt.Errorf("general account for party(%s) uses '%s' asset, but '%s' was expected",
		party, actualAsset, expectedAsset,
	)
}

func parseAccountBalancesTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"asset",
	}, []string{
		"market id",
		"margin",
		"general",
		"bond",
		"vesting",
		"vested",
		"order margin",
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

func (r accountBalancesRow) OrderMarginAccountBalance() string {
	return r.row.MustStr("order margin")
}

func (r accountBalancesRow) GeneralAccountBalance() string {
	return r.row.MustStr("general")
}

func (r accountBalancesRow) ExpectBondAccountBalance() bool {
	return r.row.HasColumn("bond")
}

func (r accountBalancesRow) ExpectGeneralAccountBalance() bool {
	return r.row.HasColumn("general")
}

func (r accountBalancesRow) ExpectMarginAccountBalance() bool {
	return r.row.HasColumn("margin")
}

func (r accountBalancesRow) ExpectOrderMarginAccountBalance() bool {
	return r.row.HasColumn("order margin")
}

func (r accountBalancesRow) ExpectAsset() bool {
	return r.row.HasColumn("asset")
}

func (r accountBalancesRow) ExpectMarketID() bool {
	return r.row.HasColumn("market id")
}

func (r accountBalancesRow) BondAccountBalance() string {
	return r.row.MustStr("bond")
}

func (r accountBalancesRow) VestedAccountBalance() string {
	return r.row.MustStr("vested")
}

func (r accountBalancesRow) ExpectVestedAccountBalance() bool {
	return r.row.HasColumn("vested")
}

func (r accountBalancesRow) VestingAccountBalance() string {
	return r.row.MustStr("vesting")
}

func (r accountBalancesRow) ExpectVestingAccountBalance() bool {
	return r.row.HasColumn("vesting")
}
