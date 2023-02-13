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

package steps

import (
	"fmt"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/integration/stubs"
)

func PartiesShouldHaveTheFollowingAccountBalances(
	broker *stubs.BrokerStub,
	table *godog.Table,
) error {
	for _, r := range parseAccountBalancesTable(table) {
		row := accountBalancesRow{row: r}
		var hasError bool

		var actGenAccBal, actMarAccBal, actBondAccBal string
		expectedAsset := row.Asset()
		if row.ExpectGeneralAccountBalance() && len(row.GeneralAccountBalance()) > 0 {
			generalAccount, err := broker.GetPartyGeneralAccount(row.Party(), expectedAsset)
			if err != nil {
				if row.GeneralAccountBalance() == "0" {
					// we coulnd't get the account but the expectation is 0 so it's fine
					continue
				}
				return errCannotGetPartyGeneralAccount(row.Party(), expectedAsset, err)
			}
			if generalAccount.GetAsset() != expectedAsset {
				return errWrongGeneralAccountAsset(row.Party(), expectedAsset, generalAccount.GetAsset())
			}
			actGenAccBal = generalAccount.GetBalance()
			if actGenAccBal != row.GeneralAccountBalance() {
				hasError = true
			}
		}

		if row.ExpectMarginAccountBalance() && len(row.MarginAccountBalance()) > 0 {
			if !row.ExpectMarketID() {
				return fmt.Errorf("market id must be specified when expected margin account balance is supplied")
			}
			marginAccount, err := broker.GetPartyMarginAccount(row.Party(), row.MarketID())
			if err != nil {
				if row.MarginAccountBalance() == "0" {
					// we coulnd't get the account but the expectation is 0 so it's fine
					continue
				}
				return errCannotGetPartyMarginAccount(row.Party(), row.MarketID(), err)
			}
			if marginAccount.GetAsset() != expectedAsset {
				return errWrongMarketAccountAsset(marginAccount.GetType().String(), row.Party(), row.MarketID(), expectedAsset, marginAccount.GetAsset())
			}
			actMarAccBal = marginAccount.GetBalance()
			if actMarAccBal != row.MarginAccountBalance() {
				hasError = true
			}
		}

		// check bond
		if row.ExpectBondAccountBalance() && len(row.BondAccountBalance()) > 0 {
			if !row.ExpectMarketID() {
				return fmt.Errorf("market id must be specified when expected bond account balance is supplied")
			}
			bondAcc, err := broker.GetPartyBondAccountForMarket(row.Party(), expectedAsset, row.MarketID())
			if err != nil {
				if row.BondAccountBalance() == "0" {
					// we coulnd't get the account but the expectation is 0 so it's fine
					continue
				}
				return errCannotGetPartyBondAccount(row.Party(), row.MarketID(), err)
			}
			if bondAcc.GetAsset() != expectedAsset {
				return errWrongMarketAccountAsset(bondAcc.GetType().String(), row.Party(), row.MarketID(), expectedAsset, bondAcc.GetAsset())
			}
			actBondAccBal = bondAcc.GetBalance()
			if actBondAccBal != row.BondAccountBalance() {
				hasError = true
			}
		}

		if hasError {
			return errMismatchedAccountBalances(row, actMarAccBal, actGenAccBal, actBondAccBal)
		}
	}
	return nil
}

func errCannotGetPartyGeneralAccount(party, asset string, err error) error {
	return fmt.Errorf("couldn't get general account for party(%s) and asset(%s): %s",
		party, asset, err.Error(),
	)
}

func errCannotGetPartyMarginAccount(party, market string, err error) error {
	return fmt.Errorf("couldn't get margin account for party(%s) and market(%s): %s",
		party, market, err.Error(),
	)
}

func errCannotGetPartyBondAccount(party, market string, err error) error {
	return fmt.Errorf("couldn't get bond account for party(%s) and market(%s): %s",
		party, market, err.Error(),
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

func errMismatchedAccountBalances(row accountBalancesRow, marginAccountBal, generalAccountBal, bondAccBal string) error {
	var expMarginAccountBal, expGeneralAccountBal, expBondAccountBal string
	if row.ExpectGeneralAccountBalance() {
		expGeneralAccountBal = row.GeneralAccountBalance()
	}
	if row.ExpectMarginAccountBalance() {
		expMarginAccountBal = row.MarginAccountBalance()
	}
	if row.ExpectBondAccountBalance() {
		expBondAccountBal = row.BondAccountBalance()
	}

	return formatDiff(
		fmt.Sprintf("account balances did not match for party(%s)", row.Party()),
		map[string]string{
			"margin account balance":  expMarginAccountBal,
			"general account balance": expGeneralAccountBal,
			"bond account balance":    expBondAccountBal,
		},
		map[string]string{
			"margin account balance":  marginAccountBal,
			"general account balance": generalAccountBal,
			"bond account balance":    bondAccBal,
		},
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

func (r accountBalancesRow) ExpectGeneralAccountBalance() bool {
	return r.row.HasColumn("general")
}

func (r accountBalancesRow) ExpectMarginAccountBalance() bool {
	return r.row.HasColumn("margin")
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
