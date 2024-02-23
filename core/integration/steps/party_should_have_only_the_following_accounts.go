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
	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func PartyShouldHaveOnlyTheFollowingAccounts(
	broker *stubs.BrokerStub,
	owner string,
	table *godog.Table,
) error {
	// Get all the accounts and filter out just the ones for this party
	accounts := broker.GetAccounts()
	samePartyAccounts := make([]types.Account, 0)
	for _, acc := range accounts {
		if acc.Owner == owner {
			samePartyAccounts = append(samePartyAccounts, acc)
		}
	}

	suppliedRows := parseAccountTypeAndAssetTable(table)

	// Check we have the same number of rows in each set
	if len(samePartyAccounts) != len(suppliedRows) {
		return fmt.Errorf("the number of rows in the table (%v) does not match the number of accounts for that party (%v)",
			len(suppliedRows), len(samePartyAccounts))
	}

	// Go through every supplied row and make sure it matches one of the account rows
	for _, r := range suppliedRows {
		row := accountTypeAndAssetRow{row: r}
		found := false
		for _, acc := range samePartyAccounts {
			if row.Type() == acc.Type.Enum().String() && row.Asset() == acc.Asset && (len(row.Amount()) == 0 || row.Amount() == acc.Balance) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unable to find a match to the row asset:%v, type:%v, amount:%v", row.Asset(), row.Type(), row.Amount())
		}
	}
	return nil
}

func parseAccountTypeAndAssetTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"type",
		"asset",
	}, []string{
		"amount",
	})
}

type accountTypeAndAssetRow struct {
	row RowWrapper
}

func (r accountTypeAndAssetRow) Type() string {
	return r.row.MustStr("type")
}

func (r accountTypeAndAssetRow) Asset() string {
	return r.row.MustStr("asset")
}

func (r accountTypeAndAssetRow) Amount() string {
	if r.row.HasColumn("amount") {
		return r.row.MustUint("amount").String()
	}
	return ""
}
