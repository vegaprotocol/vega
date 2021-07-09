package steps

import (
	"fmt"

	"code.vegaprotocol.io/data-node/integration/stubs"
	"code.vegaprotocol.io/data-node/logging"
)

func DebugTrades(broker *stubs.BrokerStub, log *logging.Logger) error {
	log.Info("DUMPING TRADES")
	data := broker.GetTrades()
	for _, t := range data {
		log.Info(fmt.Sprintf("trade %s, %#v\n", t.Id, t.String()))
	}
	return nil
}
