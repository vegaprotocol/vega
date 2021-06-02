package steps

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"github.com/cucumber/godog/gherkin"
)

func TradersPlaceTheFollowingPeggedOrders(exec *execution.Engine, orders *gherkin.DataTable) error {
	for i, row := range TableWrapper(*orders).Parse() {
		trader := row.MustStr("trader")
		marketID := row.MustStr("market id")
		side := row.MustSide("side")
		volume := row.MustU64("volume")
		reference := row.MustPeggedReference("reference")
		offset := row.MustI64("offset")

		orderSubmission := &commandspb.OrderSubmission{
			Type:        types.Order_TYPE_LIMIT,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Side:        side,
			MarketId:    marketID,
			Size:        volume,
			Reference:   fmt.Sprintf("%s-pegged-order-%d", trader, i),
			PeggedOrder: &types.PeggedOrder{
				Reference: reference,
				Offset:    offset,
			},
		}
		_, err := exec.SubmitOrder(context.Background(), orderSubmission, trader)
		if err != nil {
			if row.Has("error") {
				if err.Error() == row.MustStr("error") {
					continue
				}
				return fmt.Errorf("expected error '%s', instead got '%s'", row.MustStr("error"), err.Error())
			}
			return errSubmitOrder(err, orderSubmission)
		}
	}
	return nil
}

func errSubmitOrder(err error, o *commandspb.OrderSubmission) error {
	return fmt.Errorf("error submitting order [%v]: %v", o, err)
}
