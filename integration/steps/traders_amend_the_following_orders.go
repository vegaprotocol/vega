package steps

import (
	"context"
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/types"
)

type OrderAmendmentError struct {
	OrderAmendment types.OrderAmendment
	OrderReference string
	Err            error
}

func (o OrderAmendmentError) Error() string {
	return fmt.Sprintf("%v: %v", o.OrderAmendment, o.Err)
}

func TradersAmendTheFollowingOrders(
	broker *stubs.BrokerStub,
	exec *execution.Engine,
	table *gherkin.DataTable,
) error {
	for _, r := range parseAmendOrderTable(table) {
		row := amendOrderRow{row: r}

		o, err := broker.GetByReference(row.Party(), row.Reference())
		if err != nil {
			return errOrderNotFound(row.Reference(), row.Party(), err)
		}

		amend := types.OrderAmendment{
			OrderId:     o.Id,
			MarketId:    o.MarketId,
			SizeDelta:   row.SizeDelta(),
			TimeInForce: row.TimeInForce(),
		}

		if row.Price() != nil {
			amend.Price = row.Price().Value
		}

		if row.ExpirationDate() != nil {
			amend.ExpiresAt = &row.ExpirationDate().Value
		}

		_, err = exec.AmendOrder(context.Background(), &amend, o.PartyId)
		if err := checkExpectedError(row, err); err != nil {
			return err
		}
	}

	return nil
}

type amendOrderRow struct {
	row RowWrapper
}

func parseAmendOrderTable(table *gherkin.DataTable) []RowWrapper {
	return StrictParseTable(table, []string{
		"trader",
		"reference",
		"price",
		"size delta",
		"tif",
	}, []string{
		"error",
		"expiration date",
	})
}

func (r amendOrderRow) Party() string {
	return r.row.MustStr("trader")
}

func (r amendOrderRow) Reference() string {
	return r.row.MustStr("reference")
}

func (r amendOrderRow) Price() *types.Price {
	return r.row.MustPrice("price")
}

func (r amendOrderRow) SizeDelta() int64 {
	return r.row.MustI64("size delta")
}

func (r amendOrderRow) TimeInForce() types.Order_TimeInForce {
	return r.row.MustTIF("tif")
}

func (r amendOrderRow) ExpirationDate() *types.Timestamp {
	if !r.row.HasColumn("expiration date") {
		return nil
	}

	timeNano := r.row.MustTime("expiration date").UnixNano()
	if timeNano == 0 {
		return nil
	}

	return &types.Timestamp{Value: timeNano}
}

func (r amendOrderRow) Error() string {
	return r.row.Str("error")
}

func (r amendOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}
