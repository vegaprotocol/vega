package steps

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TradersAmendOrdersByReference(broker *stubs.BrokerStub, exec *execution.Engine, table *gherkin.DataTable) error {
	for _, row := range TableWrapper(*table).Parse() {
		trader := row.Str("trader")
		reference := row.Str("reference")
		price := row.Price("price")
		sizeDelta := row.I64("sizeDelta")
		tif := row.TIF("tif")
		success := row.Bool("success")

		o, err := broker.GetByReference(trader, reference)
		if err != nil {
			return errOrderNotFound(reference, trader, err)
		}

		amend := types.OrderAmendment{
			OrderId:     o.Id,
			PartyId:     o.PartyId,
			MarketId:    o.MarketId,
			Price:       price,
			SizeDelta:   sizeDelta,
			TimeInForce: tif,
		}

		_, err = exec.AmendOrder(context.Background(), &amend)
		if err != nil && success {
			return errAmendingOrder(o, err)
		}

		if err == nil && !success {
			return fmt.Errorf("expected to failed amending but succeed for trader %s (reference %s)", o.PartyId, o.Reference)
		}

	}

	return nil
}

func errAmendingOrder(o types.Order, err error) error {
	return fmt.Errorf("expected to succeed amending but failed for trader %s (reference %s, err %v)", o.PartyId, o.Reference, err)
}
