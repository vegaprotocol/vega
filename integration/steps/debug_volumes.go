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

func DebugVolumesForMarketDetail(log *logging.Logger, broker *stubs.BrokerStub, marketID string) error {
	sell, buy := broker.GetActiveOrderDepth(marketID)
	s := fmt.Sprintf("\nSELL orders:\n\t|%20s |%10s |%10s |%40s |", "Party", "Volume", "Remaining", "Price")
	for _, o := range sell {
		s += fmt.Sprintf("\n\t|%20s |%10d |%10d |%40s |", o.PartyId, o.Size, o.Remaining, o.Price)
	}
	s += fmt.Sprintf("\nBUY orders:\n\t|%20s |%10s |%10s |%40s |", "Party", "Volume", "Remaining", "Price")
	for _, o := range buy {
		s += fmt.Sprintf("\n\t|%20s |%10d |%10d |%40s |", o.PartyId, o.Size, o.Remaining, o.Price)
	}
	log.Infof("%s\n", s)
	return nil
}
