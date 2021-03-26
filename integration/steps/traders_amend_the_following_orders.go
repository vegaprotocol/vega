package steps

import (
	"context"
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/helpers"
	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
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
	errHandler *helpers.ErrorHandler,
	broker *stubs.BrokerStub,
	exec *execution.Engine,
	table *gherkin.DataTable,
) error {
	for _, r := range TableWrapper(*table).Parse() {
		row := amendOrderRow{row: r}

		o, err := broker.GetByReference(row.trader(), row.reference())
		if err != nil {
			return errOrderNotFound(row.reference(), row.trader(), err)
		}

		amend := types.OrderAmendment{
			OrderId:     o.Id,
			PartyId:     o.PartyId,
			MarketId:    o.MarketId,
			Price:       row.price(),
			SizeDelta:   row.sizeDelta(),
			TimeInForce: row.timeInForce(),
			ExpiresAt:   row.expirationDate(),
		}

		_, err = exec.AmendOrder(context.Background(), &amend)
		if err != nil {
			errHandler.HandleError(OrderAmendmentError{
				OrderAmendment: amend,
				OrderReference: row.reference(),
				Err:            err,
			})
		}
	}

	return nil
}

type amendOrderRow struct {
	row RowWrapper
}

func (r amendOrderRow) trader() string {
	return r.row.Str("trader")
}
func (r amendOrderRow) reference() string {
	return r.row.Str("reference")
}
func (r amendOrderRow) price() *types.Price {
	return r.row.Price("price")
}
func (r amendOrderRow) sizeDelta() int64 {
	return r.row.I64("size delta")
}
func (r amendOrderRow) timeInForce() types.Order_TimeInForce {
	return r.row.TIF("tif")
}

func (r amendOrderRow) expirationDate() *types.Timestamp {
	if len(r.row.Str("expiration date")) == 0 {
		return nil
	}

	timeNano := r.row.Time("expiration date").UnixNano()
	if timeNano == 0 {
		return nil
	}

	return &types.Timestamp{Value: timeNano}
}
