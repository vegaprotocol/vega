package steps

import (
	"fmt"

	"code.vegaprotocol.io/data-node/integration/stubs"
	"code.vegaprotocol.io/data-node/logging"
)

func DebugOrders(broker *stubs.BrokerStub, log *logging.Logger) error {
	log.Info("DUMPING ORDERS")
	data := broker.GetOrderEvents()
	for _, v := range data {
		o := *v.Order()
		log.Info(fmt.Sprintf("order %s: %v\n", o.Id, o.String()))
	}
	return nil
}
