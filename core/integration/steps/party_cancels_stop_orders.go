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
