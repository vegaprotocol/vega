package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TheFollowingOrdersShouldBeStopped(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	var orderNotStopped []string
	count := len(table.Rows) - 1
	for _, row := range parseStoppedOrdersTable(table) {
		trader := row.MustStr("trader")
		marketID := row.MustStr("market id")
		reason := row.MustStr("reason")

		data := broker.GetOrderEvents()
		for _, o := range data {
			v := o.Order()
			if v.PartyId == trader && v.MarketId == marketID {
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

func parseStoppedOrdersTable(table *gherkin.DataTable) []RowWrapper {
	return TableWrapper(*table).StrictParse([]string{
		"trader",
		"market id",
		"reason",
	}, []string{})
}
