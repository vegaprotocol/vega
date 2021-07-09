package steps

import "code.vegaprotocol.io/vega/integration/stubs"

func ClearTransferEvents(broker *stubs.BrokerStub) {
	broker.ClearTransferEvents()
}
