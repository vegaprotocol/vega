package steps

import (
	"context"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/helpers"
	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/cucumber/godog/gherkin"
)

func TradersCancelTheFollowingOrders(
	broker *stubs.BrokerStub,
	exec *execution.Engine,
	errorHandler *helpers.ErrorHandler,
	orders *gherkin.DataTable,
) error {
	for _, row := range TableWrapper(*orders).Parse() {
		trader := row.Str("trader")
		reference := row.Str("reference")
		marketID := row.Str("market id")

		var orders []types.Order
		switch {
		case marketID != "":
			orders = broker.GetOrdersByPartyAndMarket(trader, marketID)
		default:
			o, err := broker.GetByReference(trader, reference)
			if err != nil {
				return errOrderNotFound(trader, reference, err)
			}
			orders = append(orders, o)
		}

		for _, o := range orders {
			cancel := types.OrderCancellation{
				OrderId:  o.Id,
				PartyId:  o.PartyId,
				MarketId: o.MarketId,
			}
			reference = o.Reference
			cancelOrder(exec, errorHandler, cancel, reference)
		}

	}

	return nil
}

func cancelOrder(exec *execution.Engine, errHandler *helpers.ErrorHandler, cancel types.OrderCancellation, ref string) {
	if _, err := exec.CancelOrder(context.Background(), &cancel); err != nil {
		errHandler.HandleError(CancelOrderError{
			reference: ref,
			request:   cancel,
			Err:       err,
		})
	}
}
