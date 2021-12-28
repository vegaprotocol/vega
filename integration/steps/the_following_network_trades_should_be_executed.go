package steps

import (
	"fmt"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/integration/stubs"
	"github.com/cucumber/godog"
)

func TheFollowingNetworkTradesShouldBeExecuted(broker *stubs.BrokerStub, table *godog.Table) error {
	for _, row := range parseNetworkTradesTable(table) {
		var (
			party         = row.MustStr("party")
			aggressorSide = row.MustSide("aggressor side")
			volume        = row.MustU64("volume")
		)

		ok := false
		data := broker.GetTrades()
		for _, v := range data {
			if (v.Buyer == party || v.Seller == party) && v.Aggressor == aggressorSide && v.Size == volume {
				ok = true
				break
			}
		}

		if !ok {
			return errTradeMissing(party, aggressorSide, volume)
		}
	}

	return nil
}

func errTradeMissing(party string, aggressorSide types.Side, volume uint64) error {
	return fmt.Errorf("expecting trade was missing: %v, %v, %v", party, aggressorSide, volume)
}

func parseNetworkTradesTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"aggressor side",
		"volume",
	}, []string{})
}
