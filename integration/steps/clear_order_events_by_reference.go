package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	"github.com/cucumber/godog/gherkin"
)

func ClearOrdersByReference(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	for _, row := range TableWrapper(*table).Parse() {
		trader := row.MustStr("trader")
		reference := row.MustStr("reference")
		if err := broker.ClearOrderByReference(trader, reference); err != nil {
			return errClearingOrder(trader, reference, err)
		}
	}
	return nil
}

func errClearingOrder(trader, reference string, err error) error {
	return fmt.Errorf("failed to clear order for trader %s with reference %s: %v", trader, reference, err)
}
