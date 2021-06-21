package steps

import (
	"context"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
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
		reference := row.Reference()
		marketID := row.MarketID()

		var orders []types.Order
		switch {
		case marketID != "":
			orders = broker.GetOrdersByPartyAndMarket(party, marketID)
		default:
			o, err := broker.GetByReference(party, reference)
			if err != nil {
				return errOrderNotFound(party, reference, err)
			}
			orders = append(orders, o)
		}

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

type cancelOrderRow struct {
	row RowWrapper
}

func parseCancelOrderTable(table *gherkin.DataTable) []RowWrapper {
	return TableWrapper(*table).StrictParse([]string{
		"trader",
	}, []string{
		"reference",
		"market id",
		"error",
	})
}

func (r cancelOrderRow) Party() string {
	return r.row.MustStr("trader")
}

func (r cancelOrderRow) MarketID() string {
	return r.row.Str("market id")
}

func (r cancelOrderRow) Reference() string {
	return r.row.Str("reference")
}

func (r cancelOrderRow) Error() string {
	return r.row.Str("error")
}

func (r cancelOrderRow) ExpectError() bool {
	return len(r.row.Str("error")) > 0
}
