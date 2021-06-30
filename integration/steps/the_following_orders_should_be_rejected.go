package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TheFollowingOrdersShouldBeRejected(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	var orderNotRejected []string
	count := len(table.Rows) - 1
	for _, row := range parseRejectedOrdersTable(table) {
		trader := row.MustStr("trader")
		marketID := row.MustStr("market id")
		reason := row.MustStr("reason")

		data := broker.GetOrderEvents()
		for _, o := range data {
			v := o.Order()
			if v.PartyId == trader && v.MarketId == marketID {
				if v.Status == types.Order_STATUS_REJECTED && v.Reason.String() == reason {
					count -= 1
					continue
				}
				orderNotRejected = append(orderNotRejected, v.Reference)
			}
		}
	}

	if count > 0 {
		return errOrderNotRejected(orderNotRejected)
	}

	return nil
}

func errOrderNotRejected(orderNotRejected []string) error {
	return fmt.Errorf("orders with reference %v were not rejected", orderNotRejected)
}

func parseRejectedOrdersTable(table *gherkin.DataTable) []RowWrapper {
	return StrictParseTable(table, []string{
		"trader",
		"market id",
		"reason",
	}, []string{})
}
