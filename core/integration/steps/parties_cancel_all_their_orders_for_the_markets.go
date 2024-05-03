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
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"

	"github.com/cucumber/godog"
)

func PartiesCancelAllTheirOrdersForTheMarkets(
	broker *stubs.BrokerStub,
	exec Execution,
	table *godog.Table,
) error {
	for _, r := range parseCancelAllOrderTable(table) {
		row := cancelAllOrderRow{row: r}
		party := row.Party()
		cancel := types.OrderCancellation{
			MarketID: row.MarketID(),
		}
		_, err := exec.CancelOrder(context.Background(), &cancel, party)
		err = checkExpectedError(row, err, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

type cancelAllOrderRow struct {
	row RowWrapper
}

func parseCancelAllOrderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
	}, []string{
		"error",
	})
}

func (r cancelAllOrderRow) Party() string {
	return r.row.MustStr("party")
}

func (r cancelAllOrderRow) MarketID() string {
	return r.row.Str("market id")
}

func (r cancelAllOrderRow) Reference() string {
	return fmt.Sprintf("%s-%s", r.Party(), r.MarketID())
}

func (r cancelAllOrderRow) Error() string {
	return r.row.Str("error")
}

func (r cancelAllOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}
