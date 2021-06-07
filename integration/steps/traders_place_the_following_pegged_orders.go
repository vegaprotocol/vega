package steps

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	ptypes "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types"

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
			PeggedOrder: &ptypes.PeggedOrder{
				Reference: reference,
				Offset:    offset,
			},
		}
		_, err := exec.SubmitOrder(context.Background(), orderSubmission, trader)
		if err != nil {
			return errSubmitOrder(err, orderSubmission)
		}
	}
	return nil
}

func errSubmitOrder(err error, o *commandspb.OrderSubmission) error {
	return fmt.Errorf("error submitting order [%v]: %v", o, err)
}
