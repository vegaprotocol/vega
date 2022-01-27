package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
)

func DebugVolumesForMarket(log *logging.Logger, broker *stubs.BrokerStub, marketID string) error {
	sell, buy := broker.GetBookDepth(marketID)
	log.Info("SELL volume:")
	for price, vol := range sell {
		log.Info(fmt.Sprintf("Price %s: %d\n", price, vol))
	}
	log.Info("BUY volume:")
	for price, vol := range buy {
		log.Info(fmt.Sprintf("Price %s: %d\n", price, vol))
	}
	return nil
}
