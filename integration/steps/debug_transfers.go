package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
)

func DebugTransfers(broker *stubs.BrokerStub, log *logging.Logger) error {
	log.Info("DUMPING TRANSFERS")
	transferEvents := broker.GetTransferResponses()
	for _, e := range transferEvents {
		for _, t := range e.TransferResponses() {
			for _, v := range t.GetTransfers() {
				log.Info(fmt.Sprintf("transfer: %v\n", v.String()))
			}
		}
	}
	return nil
}
