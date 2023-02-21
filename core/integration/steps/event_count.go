package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/stubs"
)

func ExpectingEventsOverStep(broker *stubs.BrokerStub, eventsBeforeStep, expected int) error {
	actual := len(broker.GetAllEvents()) - eventsBeforeStep
	if expected == actual {
		return nil
	}
	return fmt.Errorf("expecting '%d' events generated over the last step, found '%d'", expected, actual)
}

func ExpectingEventsInTheSecenarioSoFar(broker *stubs.BrokerStub, expected int) error {
	events := broker.GetAllEvents()
	actual := len(events)
	if expected == actual {
		return nil
	}
	return fmt.Errorf("expecting '%d' events generated in the scenario so far, found '%d'", expected, actual)
}
