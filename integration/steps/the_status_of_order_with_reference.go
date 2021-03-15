package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TheStatusOfOrderWithReference(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	for _, row := range TableWrapper(*table).Parse() {
		trader := row.Str("trader")
		reference := row.Str("reference")
		status := row.OrderStatus("status")

		if trader == "trader" {
			continue
		}

		o, err := broker.GetByReference(trader, reference)
		if err != nil {
			return err
		}

		if status != o.Status {
			return errInvalidOrderStatus(o, status)
		}
	}

	return nil
}

func errInvalidOrderStatus(o types.Order, status types.Order_Status) error {
	return fmt.Errorf("invalid order status for order ref %v, expected %v got %v", o.Reference, status, o.Status)
}
