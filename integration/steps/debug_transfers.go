package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func DebugTransfers(broker *stubs.BrokerStub) error {
	fmt.Println("DUMPING TRANSFERS")
	transferEvents := broker.GetTransferResponses()
	for _, e := range transferEvents {
		for _, t := range e.TransferResponses() {
			for _, v := range t.GetTransfers() {
				fmt.Printf("transfer: %v\n", *v)
			}
		}
	}
	return nil
}
