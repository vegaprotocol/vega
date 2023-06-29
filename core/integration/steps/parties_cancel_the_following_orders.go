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
		var err error

		if row.Stop() {
			order, err := broker.GetStopByReference(party, row.Reference())
			if err != nil {
				return errOrderNotFound(row.Reference(), party, err)
			}
			cancel := types.StopOrdersCancellation{
				OrderID:  order.StopOrder.Id,
				MarketID: order.StopOrder.MarketId,
			}
			err = exec.CancelStopOrder(context.Background(), &cancel, party)
		} else {
			order, err := broker.GetByReference(party, row.Reference())
			if err != nil {
				return errOrderNotFound(row.Reference(), party, err)
			}
			cancel := types.OrderCancellation{
				OrderID:  order.Id,
				MarketID: order.MarketId,
			}
			_, err = exec.CancelOrder(context.Background(), &cancel, party)
		}

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
		"stop",
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

func (r cancelOrderRow) Stop() bool {
	if !r.row.HasColumn("stop") {
		return false
	}

	return r.row.Bool("stop")
}
