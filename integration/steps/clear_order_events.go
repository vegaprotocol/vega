package steps

import "code.vegaprotocol.io/vega/integration/stubs"

func ClearOrderEvents(broker *stubs.BrokerStub) {
	broker.ClearOrderEvents()
}
