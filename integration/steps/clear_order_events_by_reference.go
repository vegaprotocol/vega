package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	"github.com/cucumber/godog"
)

func ClearOrdersByReference(broker *stubs.BrokerStub, table *godog.Table) error {
	for _, row := range parseClearOrdersTable(table) {
		party := row.MustStr("party")
		reference := row.MustStr("reference")
		if err := broker.ClearOrderByReference(party, reference); err != nil {
			return errClearingOrder(party, reference, err)
		}
	}
	return nil
}

func errClearingOrder(party, reference string, err error) error {
	return fmt.Errorf("failed to clear order for party %s with reference %s: %v", party, reference, err)
}

func parseClearOrdersTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"reference",
	}, []string{})
}
