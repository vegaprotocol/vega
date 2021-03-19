package steps

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TradersCancelsTheFollowingOrders(broker *stubs.BrokerStub, exec *execution.Engine, orders *gherkin.DataTable) error {
	for _, row := range TableWrapper(*orders).Parse() {
		trader := row.Str("trader")
		reference := row.Str("reference")

		o, err := broker.GetByReference(trader, reference)
		if err != nil {
			return err
		}

		cancel := types.OrderCancellation{
			OrderId:  o.Id,
			PartyId:  o.PartyId,
			MarketId: o.MarketId,
		}

		if _, err = exec.CancelOrder(context.Background(), &cancel); err == nil {
			return fmt.Errorf("successfully cancelled order for trader %s (reference %s)", o.PartyId, o.Reference)
		}
	}

	return nil
}
