package steps

import (
	"code.vegaprotocol.io/vega/integration/stubs"
	"github.com/cucumber/godog/gherkin"
)

func ClearOrdersByReference(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	for _, row := range TableWrapper(*table).Parse() {
		trader := row.Str("trader")
		reference := row.Str("reference")
		if trader == "trader" {
			continue
		}
		if err := broker.ClearOrderByReference(trader, reference); err != nil {
			return err
		}
	}
	return nil
}
