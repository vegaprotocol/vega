package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
)

func DebugAuctionEvents(broker *stubs.BrokerStub, log *logging.Logger) error {
	log.Info("DUMPING AUCTION EVENTS")
	data := broker.GetAuctionEvents()
	for _, a := range data {
		log.Info(fmt.Sprintf("AuctionEvent summary: %s, %#v\n", a.MarketEvent(), a))
	}
	return nil
}
