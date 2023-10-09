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

func PartiesCancelTheFollowingStopOrders(
	broker *stubs.BrokerStub,
	exec Execution,
	table *godog.Table,
) error {
	for _, r := range parseCancelStopOrderTable(table) {
		row := cancelStopOrderRow{row: r}

		party := row.Party()
		var err error

		order, err := broker.GetStopByReference(party, row.Reference())
		if err != nil {
			return errOrderNotFound(row.Reference(), party, err)
		}
		cancel := types.StopOrdersCancellation{
			OrderID:  order.StopOrder.Id,
			MarketID: order.StopOrder.MarketId,
		}
		err = exec.CancelStopOrder(context.Background(), &cancel, party)

		err = checkExpectedError(row, err, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func PartyCancelsAllTheirStopOrders(
	exec Execution,
	partyID string,
) error {
	cancel := types.StopOrdersCancellation{
		OrderID:  "",
		MarketID: "",
	}
	_ = exec.CancelStopOrder(context.Background(), &cancel, partyID)
	return nil
}

func PartyCancelsAllTheirStopOrdersForTheMarket(
	exec Execution,
	partyID string,
	marketID string,
) error {
	cancel := types.StopOrdersCancellation{
		OrderID:  "",
		MarketID: marketID,
	}
	_ = exec.CancelStopOrder(context.Background(), &cancel, partyID)
	return nil
}

type cancelStopOrderRow struct {
	row RowWrapper
}

func parseCancelStopOrderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"reference",
	}, []string{
		"error",
	})
}

func (r cancelStopOrderRow) Party() string {
	return r.row.MustStr("party")
}

func (r cancelStopOrderRow) Reference() string {
	return r.row.Str("reference")
}

func (r cancelStopOrderRow) Error() string {
	return r.row.Str("error")
}

func (r cancelStopOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}
