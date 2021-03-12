package steps

import (
	"context"
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func TradersAttemptToCancelTheFollowingFilledOrders(
	broker *stubs.BrokerStub,
	exec *execution.Engine,
	table *gherkin.DataTable,
) error {
	for _, row := range TableWrapper(*table).Parse() {
		trader := row.Str("trader")
		reference := row.Str("reference")

		o, err := broker.GetByReference(trader, reference)
		if err != nil {
			return errCannotGetOrderForParty(trader, reference, err)
		}

		cancel := types.OrderCancellation{
			OrderId:  o.Id,
			PartyId:  o.PartyId,
			MarketId: o.MarketId,
		}

		if _, err = exec.CancelOrder(context.Background(), &cancel); err == nil {
			return errCanceledFilledOrder(o)
		}
	}

	return nil
}

func errCannotGetOrderForParty(partyID, reference string, err error) error {
	return fmt.Errorf("couldn't get order for party(%s) and reference(%s): %s", partyID, reference, err.Error())
}

func errCanceledFilledOrder(o types.Order) error {
	return fmt.Errorf("trader(%s) succesfully canceled an uncancelable order with reference(%s)",
		o.PartyId, o.Reference,
	)
}
