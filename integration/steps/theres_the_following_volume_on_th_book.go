package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/cucumber/godog/gherkin"
)

func TheresTheFollowingVolumeOnTheBook(
	broker *stubs.BrokerStub,
	table *gherkin.DataTable,
) error {
	for _, row := range TableWrapper(*table).Parse() {
		market := row.Str("market id")
		volume := row.U64("volume")
		price := row.U64("price")
		side := row.Side("side")

		sell, buy := broker.GetBookDepth(market)
		if side == types.Side_SIDE_SELL {
			vol := sell[price]
			if vol != volume {
				return fmt.Errorf("invalid volume(%d) at price(%d) and side(%s) for market(%v), expected(%v)", vol, price, side.String(), market, volume)
			}
			continue
		}
		vol := buy[price]
		if vol != volume {
			return fmt.Errorf("invalid volume(%d) at price(%d) and side(%s) for market(%v), expected(%v)", vol, price, side.String(), market, volume)
		}
	}
	return nil
}
