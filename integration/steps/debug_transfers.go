package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
)

func DebugTransfers(broker *stubs.BrokerStub, log *logging.Logger) {
	log.Info("DUMPING TRANSFERS")
	s := fmt.Sprintf("\n\t|%40s |%40s |%25s |%25s |%15s |\n", "Type", "Reference", "From", "To", "Amount")
	transferEvents := broker.GetTransferResponses()
	for _, e := range transferEvents {
		for _, t := range e.TransferResponses() {
			for _, v := range t.GetTransfers() {
				s += fmt.Sprintf("\t|%40s |%40s |%25s |%25s |%15s |\n", v.Type, v.Reference, v.FromAccount, v.ToAccount, v.Amount)
			}
		}
	}
	log.Info(s)
}
