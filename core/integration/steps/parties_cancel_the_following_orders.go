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

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"

	"github.com/cucumber/godog"
)

func PartiesCancelTheFollowingOrders(
	broker *stubs.BrokerStub,
	exec Execution,
	table *godog.Table,
) error {
	for _, r := range parseCancelOrderTable(table) {
		row := cancelOrderRow{row: r}

		party := row.Party()

		order, err := broker.GetByReference(party, row.Reference())
		if err != nil {
			return errOrderNotFound(row.Reference(), party, err)
		}

		cancel := types.OrderCancellation{
			OrderID:  order.Id,
			MarketID: order.MarketId,
		}

		_, err = exec.CancelOrder(context.Background(), &cancel, party)
		err = checkExpectedError(row, err, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

type cancelOrderRow struct {
	row RowWrapper
}

func parseCancelOrderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"reference",
	}, []string{
		"error",
	})
}

func (r cancelOrderRow) Party() string {
	return r.row.MustStr("party")
}

func (r cancelOrderRow) HasMarketID() bool {
	return r.row.HasColumn("market id")
}

func (r cancelOrderRow) Reference() string {
	return r.row.Str("reference")
}

func (r cancelOrderRow) Error() string {
	return r.row.Str("error")
}

func (r cancelOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}
