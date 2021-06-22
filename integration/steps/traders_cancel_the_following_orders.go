package steps

import (
	"context"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/stubs"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"

	"github.com/cucumber/godog/gherkin"
)

func TradersCancelTheFollowingOrders(
	broker *stubs.BrokerStub,
	exec *execution.Engine,
	table *gherkin.DataTable,
) error {
	for _, r := range parseCancelOrderTable(table) {
		row := cancelOrderRow{row: r}

		party := row.Party()

		order, err := broker.GetByReference(party, row.Reference())
		if err != nil {
			return errOrderNotFound(party, row.Reference(), err)
		}

		cancel := commandspb.OrderCancellation{
			OrderId:  order.Id,
			MarketId: order.MarketId,
		}

		_, err = exec.CancelOrder(context.Background(), &cancel, party)
		err = checkExpectedError(row, err)
		if err != nil {
			return err
		}

	}

	return nil
}

type cancelOrderRow struct {
	row RowWrapper
}

func parseCancelOrderTable(table *gherkin.DataTable) []RowWrapper {
	return TableWrapper(*table).StrictParse([]string{
		"trader",
		"reference",
	}, []string{
		"error",
	})
}

func (r cancelOrderRow) Party() string {
	return r.row.MustStr("trader")
}

func (r cancelOrderRow) HasMarketID() bool {
	return r.row.HasColumn("market id")
}

func (r cancelOrderRow) Reference() string {
	return r.row.Str("reference")
}

func (r cancelOrderRow) Error() string {
	return r.row.Str("error")
}

func (r cancelOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}
