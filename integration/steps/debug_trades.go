package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func DebugTrades(broker *stubs.BrokerStub) error {
	fmt.Println("DUMPING TRADES")
	data := broker.GetTrades()
	for _, t := range data {
		fmt.Printf("trade %s, %#v\n", t.Id, t)
	}
	return nil
}
