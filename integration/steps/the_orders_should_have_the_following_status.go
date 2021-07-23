package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog"
)

func TheOrdersShouldHaveTheFollowingStatus(broker *stubs.BrokerStub, table *godog.Table) error {
	for _, row := range parseOrderStatusTable(table) {
		party := row.MustStr("party")
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

func parseOrderStatusTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"reference",
		"status",
	}, []string{})
}
