package steps

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TradersCancelPeggedOrders(broker *stubs.BrokerStub, exec *execution.Engine, orders *gherkin.DataTable) error {
	cancellations := make([]types.OrderCancellation, 0, len(orders.Rows))
	for _, row := range TableWrapper(*orders).Parse() {
		trader := row.Str("trader")
		marketID := row.Str("market id")

		orders := broker.GetOrdersByPartyAndMarket(trader, marketID)
		if len(orders) == 0 {
			return errPeggedOrdersNotFound(trader, marketID)
		}

		// orders have to be pegged:
		found := false
		for _, o := range orders {
			if o.PeggedOrder != nil && o.Status != types.Order_STATUS_CANCELLED && o.Status != types.Order_STATUS_REJECTED {
				cancellations = append(cancellations, types.OrderCancellation{
					PartyId:  trader,
					MarketId: marketID,
					OrderId:  o.Id,
				})
				found = true
				break
			}
		}
		if !found {
			return errPeggedOrdersNotFound(trader, marketID)
		}
	}

	broker.ClearOrderEvents()

	for _, c := range cancellations {
		if _, err := exec.CancelOrder(context.Background(), &c); err != nil {
			return errPeggedOrderCancellationFailed(c)
		}
	}
	return nil
}

func errPeggedOrderCancellationFailed(c types.OrderCancellation) error {
	return fmt.Errorf("failed to cancel pegged order %s for %s on market %s", c.OrderId, c.PartyId, c.MarketId)
}

func errPeggedOrdersNotFound(trader string, marketID string) error {
	return fmt.Errorf("no pegged orders found for party %s on market %s", trader, marketID)
}