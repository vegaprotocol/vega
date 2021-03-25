package steps

import (
	"context"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/helpers"
	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TradersCancelsTheFollowingOrders(broker *stubs.BrokerStub, exec *execution.Engine, errorHandler *helpers.ErrorHandler, orders *gherkin.DataTable) error {
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

		_, err = exec.CancelOrder(context.Background(), &cancel)
		if err != nil {
			errorHandler.HandleError(CancelOrderError{
				reference: reference,
				request:   cancel,
				Err:       err,
			})
		}
	}

	return nil
}
