package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/integration/stubs"
	"github.com/cucumber/godog"
)

func TheFollowingEventsShouldBeEmitted(broker *stubs.BrokerStub, table *godog.Table) error {
	for _, row := range parseEmittedEventsTable(table) {
		eventType := row.MustEventType("type")

		if len(broker.GetBatch(eventType)) == 0 {
			return errEventNotEmitted(eventType)
		}
	}

	return nil
}

func parseEmittedEventsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"type",
	}, []string{})
}

func errEventNotEmitted(t events.Type) error {
	return fmt.Errorf("event of type %s has not been emitted", t)
}
