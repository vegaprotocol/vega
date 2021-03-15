package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TheFollowingNetworkTradesHappened(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	for _, row := range TableWrapper(*table).Parse() {
		var (
			trader        = row.Str("trader")
			aggressorSide = row.Side("aggressor side")
			volume        = row.U64("volume")
		)

		ok := false
		data := broker.GetTrades()
		for _, v := range data {
			if (v.Buyer == trader || v.Seller == trader) && v.Aggressor == aggressorSide && v.Size == volume {
				ok = true
				break
			}
		}

		if !ok {
			return errTradeMissing(trader, aggressorSide, volume)
		}
	}

	return nil
}

func errTradeMissing(party string, aggressorSide types.Side, volume uint64) error {
	return fmt.Errorf("expecting trade was missing: %v, %v, %v", party, aggressorSide, volume)
}
