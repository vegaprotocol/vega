package steps

import "code.vegaprotocol.io/data-node/integration/stubs"

func ClearTransferEvents(broker *stubs.BrokerStub) {
	broker.ClearTransferEvents()
}
