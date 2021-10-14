package steps

import (
	"context"

	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/types"

	"github.com/cucumber/godog"
)

func PartiesCancelTheFollowingOrders(
	broker *stubs.BrokerStub,
	exec Execution,
	table *godog.Table,
) error {
	for _, r := range parseCancelOrderTable(table) {
		row := cancelOrderRow{row: r}

		party := row.Party()

		order, err := broker.GetByReference(party, row.Reference())
		if err != nil {
			return errOrderNotFound(party, row.Reference(), err)
		}

		cancel := types.OrderCancellation{
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

func parseCancelOrderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"reference",
	}, []string{
		"error",
	})
}

func (r cancelOrderRow) Party() string {
	return r.row.MustStr("party")
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
