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
	"strconv"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"github.com/cucumber/godog"
)

func PartyShouldHaveGeneralAccountBalanceForAsset(
	broker *stubs.BrokerStub,
	party, asset, rawBalance string,
) error {
	balance, _ := strconv.ParseUint(rawBalance, 10, 0)
	acc, err := broker.GetPartyGeneralAccount(party, asset)
	if err != nil {
		return err
	}

	if stringToU64(acc.Balance) != balance {
		return fmt.Errorf("invalid general account balance for asset(%s) for party(%s), expected(%d) got(%s)",
			asset, party, balance, acc.Balance,
		)
	}

	return nil
}

func PartyShouldHaveVestingAccountBalanceForAsset(
	broker *stubs.BrokerStub,
	party, asset, rawBalance string,
) error {
	balance, _ := strconv.ParseUint(rawBalance, 10, 0)
	acc, err := broker.GetPartyVestingAccount(party, asset)
	if err != nil {
		return err
	}

	if stringToU64(acc.Balance) != balance {
		return fmt.Errorf("invalid vesting account balance for asset(%s) for party(%s), expected(%d) got(%s)",
			asset, party, balance, acc.Balance,
		)
	}

	return nil
}

func PartiesShouldHaveVestingAccountBalances(broker *stubs.BrokerStub, table *godog.Table) error {
	for _, r := range parseVestingRow(table) {
		row := vestingRow{
			r: r,
		}
		acc, err := broker.GetPartyVestingAccount(row.Party(), row.Asset())
		if err != nil {
			if err != stubs.AccountDoesNotExistErr {
				return err
			}
			acc.Balance = "0"
		}
		if stringToU64(acc.Balance) != row.Balance() {
			return fmt.Errorf("invalid vesting account balance for asset (%s) for party(%s), expected (%d) got (%s)",
				row.Asset(), row.Party(), row.Balance(), acc.Balance,
			)
		}
	}
	return nil
}

func PartyShouldHaveVestedAccountBalanceForAsset(
	broker *stubs.BrokerStub,
	party, asset, rawBalance string,
) error {
	balance, _ := strconv.ParseUint(rawBalance, 10, 0)
	acc, err := broker.GetPartyVestedAccount(party, asset)
	if err != nil {
		return err
	}

	if stringToU64(acc.Balance) != balance {
		return fmt.Errorf("invalid vested account balance for asset(%s) for party(%s), expected(%d) got(%s)",
			asset, party, balance, acc.Balance,
		)
	}

	return nil
}

func PartyShouldHaveHoldingAccountBalanceForAsset(
	broker *stubs.BrokerStub,
	party, asset, rawBalance string,
) error {
	balance, _ := strconv.ParseUint(rawBalance, 10, 0)
	acc, err := broker.GetPartyHoldingAccount(party, asset)
	if err != nil {
		return err
	}

	if stringToU64(acc.Balance) != balance {
		return fmt.Errorf("invalid holding account balance for asset(%s) for party(%s), expected(%d) got(%s)",
			asset, party, balance, acc.Balance,
		)
	}

	return nil
}

func parseVestingRow(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"asset",
		"balance",
	}, []string{})
}

type vestingRow struct {
	r RowWrapper
}

func (v vestingRow) Party() string {
	return v.r.MustStr("party")
}

func (v vestingRow) Asset() string {
	return v.r.MustStr("asset")
}

func (v vestingRow) Balance() uint64 {
	return v.r.MustU64("balance")
}
