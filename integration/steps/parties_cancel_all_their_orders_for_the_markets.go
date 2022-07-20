// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
	"fmt"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/types"

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

		orders := broker.GetOrdersByPartyAndMarket(party, row.MarketID())

		dedupOrders := map[string]vega.Order{}
		for _, o := range orders {
			dedupOrders[o.Reference] = o
		}

		for _, o := range dedupOrders {
			cancel := types.OrderCancellation{
				OrderId:  o.Id,
				MarketId: o.MarketId,
			}
			_, err := exec.CancelOrder(context.Background(), &cancel, party)
			err = checkExpectedError(row, err, nil)
			if err != nil {
				return err
			}
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
