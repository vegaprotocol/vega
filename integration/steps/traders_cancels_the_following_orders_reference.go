package steps

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TradersCancelsTheFollowingFilledOrdersReference(
	broker interface {
		GetByReference(string, string) (types.Order, error)
	},
	exec *execution.Engine,
	table *gherkin.DataTable,
) error {
	for _, row := range TableWrapper(*table).Parse() {
		o, err := broker.GetByReference(
			row.Str("trader"), row.Str("reference"))
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
