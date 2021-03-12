package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func DebugOrders(broker *stubs.BrokerStub) error {
	fmt.Println("DUMPING ORDERS")
	data := broker.GetOrderEvents()
	for _, v := range data {
		o := *v.Order()
		fmt.Printf("order %s: %v\n", o.Id, o)
	}
	return nil
}
