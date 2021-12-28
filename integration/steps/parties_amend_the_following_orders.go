package steps

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"

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

func PartiesAmendTheFollowingOrders(
	broker *stubs.BrokerStub,
	exec Execution,
	table *godog.Table,
) error {
	for _, r := range parseAmendOrderTable(table) {
		row := amendOrderRow{row: r}

		o, err := broker.GetByReference(row.Party(), row.Reference())
		if err != nil {
			return errOrderNotFound(row.Reference(), row.Party(), err)
		}

		amend := types.OrderAmendment{
			OrderID:     o.Id,
			MarketID:    o.MarketId,
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

func parseAmendOrderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
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
	return r.row.MustStr("party")
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

func (r amendOrderRow) TimeInForce() types.OrderTimeInForce {
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
