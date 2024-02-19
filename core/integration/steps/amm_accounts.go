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

func PartiesHaveTheFollowingAMMBalances(broker *stubs.BrokerStub, exec Execution, table *godog.Table) error {
	for _, r := range parseAMMAccountTable(table) {
		row := ammAccRow{
			r: r,
		}
		alias := row.alias()
		id, ok := exec.GetAMMSubAccountID(alias)
		if !ok {
			return fmt.Errorf("alias %s for AMM sub account does not exist", alias)
		}
		acc, err := broker.GetPartyGeneralAccount(id, row.asset())
		if err != nil {
			return fmt.Errorf("account alias %s (ID %s) for asset %s does not exist: %v", alias, id, row.asset(), err)
		}
		if bal := row.balance(); acc.Balance != bal {
			return fmt.Errorf("account alias %s (ID %s) for asset %s: expected balance %s - instead got %s", alias, id, row.asset(), bal, acc.Balance)
		}
	}
	return nil
}

type ammAccRow struct {
	r RowWrapper
}

func parseAMMAccountTable(table *godog.Table) []RowWrapper {
	// add party and market to make the account lookup easier
	return StrictParseTable(table, []string{
		"account alias",
		"balance",
		"asset",
	}, nil)
}

func (a ammAccRow) alias() string {
	return a.r.MustStr("account alias")
}

func (a ammAccRow) balance() string {
	return a.r.MustStr("balance")
}

func (a ammAccRow) asset() string {
	return a.r.MustStr("asset")
}
