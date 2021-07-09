package steps

import "code.vegaprotocol.io/data-node/integration/stubs"

func ClearOrderEvents(broker *stubs.BrokerStub) {
	broker.ClearOrderEvents()
}
