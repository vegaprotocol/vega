package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TheOrdersShouldHaveTheFollowingStatus(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	for _, row := range parseOrderStatusTable(table) {
		party := row.MustStr("trader")
		reference := row.MustStr("reference")
		status := row.MustOrderStatus("status")

		o, err := broker.GetByReference(party, reference)
		if err != nil {
			return errOrderNotFound(reference, party, err)
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

func parseOrderStatusTable(table *gherkin.DataTable) []RowWrapper {
	return TableWrapper(*table).StrictParse([]string{
		"trader",
		"reference",
		"status",
	}, []string{})
}
