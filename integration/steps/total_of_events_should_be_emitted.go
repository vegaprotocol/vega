package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func TotalOfEventsShouldBeEmitted(broker *stubs.BrokerStub, eventCounter int) error {
	allEventCount := broker.GetAllEvents()
	if len(allEventCount) == eventCounter {
		return nil
	}

	return fmt.Errorf("expecting %d events generated, found %d", eventCounter, allEventCount)
}
