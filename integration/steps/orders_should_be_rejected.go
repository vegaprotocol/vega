package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func OrdersAreRejected(broker *stubs.BrokerStub, orders *gherkin.DataTable) error {
	var orderNotRejected []string
	count := len(orders.Rows) - 1
	for _, row := range TableWrapper(*orders).Parse() {
		trader := row.Str("trader")
		marketID := row.Str("market id")
		reason := row.Str("reason")

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
