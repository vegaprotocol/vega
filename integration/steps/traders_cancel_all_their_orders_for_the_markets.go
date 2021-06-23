package steps

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/stubs"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"

	"github.com/cucumber/godog/gherkin"
)

func TradersCancelAllTheirOrdersForTheMarkets(
	broker *stubs.BrokerStub,
	exec *execution.Engine,
	table *gherkin.DataTable,
) error {
	for _, r := range parseCancelAllOrderTable(table) {
		row := cancelAllOrderRow{row: r}

		party := row.Party()

		orders := broker.GetOrdersByPartyAndMarket(party, row.MarketID())

		for _, o := range orders {
			cancel := commandspb.OrderCancellation{
				OrderId:  o.Id,
				MarketId: o.MarketId,
			}
			_, err := exec.CancelOrder(context.Background(), &cancel, party)
			err = checkExpectedError(row, err)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

type cancelAllOrderRow struct {
	row RowWrapper
}

func parseCancelAllOrderTable(table *gherkin.DataTable) []RowWrapper {
	return StrictParseTable(table, []string{
		"trader",
		"market id",
	}, []string{
		"error",
	})
}

func (r cancelAllOrderRow) Party() string {
	return r.row.MustStr("trader")
}

func (r cancelAllOrderRow) MarketID() string {
	return r.row.Str("market id")
}

func (r cancelAllOrderRow) Reference() string {
	return fmt.Sprintf("%s-%s", r.Party(), r.MarketID())
}

func (r cancelAllOrderRow) Error() string {
	return r.row.Str("error")
}

func (r cancelAllOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}
