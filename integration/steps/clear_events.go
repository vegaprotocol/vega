package steps

import "code.vegaprotocol.io/vega/integration/stubs"

func ClearAllEvents(broker *stubs.BrokerStub) {
	broker.ClearAllEvents()
}
