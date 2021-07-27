package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/protos/vega"
	"github.com/cucumber/godog"
)

func TheFollowingOrdersShouldBeStopped(broker *stubs.BrokerStub, table *godog.Table) error {
	var orderNotStopped []string
	count := len(table.Rows) - 1
	for _, row := range parseStoppedOrdersTable(table) {
		party := row.MustStr("party")
		marketID := row.MustStr("market id")
		reason := row.MustStr("reason")

		data := broker.GetOrderEvents()
		for _, o := range data {
			v := o.Order()
			if v.PartyId == party && v.MarketId == marketID {
				if v.Status == types.Order_STATUS_STOPPED && v.Reason.String() == reason {
					count -= 1
					continue
				}
				orderNotStopped = append(orderNotStopped, v.Reference)
			}
		}
	}

	if count > 0 {
		return errOrderNotStopped(orderNotStopped)
	}

	return nil
}

func errOrderNotStopped(orderNotRejected []string) error {
	return fmt.Errorf("orders with reference %v were not stopped", orderNotRejected)
}

func parseStoppedOrdersTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"reason",
	}, []string{})
}
