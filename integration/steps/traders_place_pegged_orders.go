package steps

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TradersPlacePeggedOrders(exec *execution.Engine, orders *gherkin.DataTable) error {
	for i, row := range TableWrapper(*orders).Parse() {
		trader := row.MustStr("trader")
		marketID := row.MustStr("market id")
		side := row.MustSide("side")
		volume := row.MustU64("volume")
		reference := row.MustPeggedReference("reference")
		offset := row.MustI64("offset")

		o := &types.Order{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Id:          "someid",
			Side:        side,
			PartyId:     trader,
			MarketId:    marketID,
			Size:        volume,
			Remaining:   volume,
			Reference:   fmt.Sprintf("%s-pegged-order-%d", trader, i),
			PeggedOrder: &types.PeggedOrder{
				Reference: reference,
				Offset:    offset,
			},
		}
		_, err := exec.SubmitOrder(context.Background(), o)
		if err != nil {
			return errSubmitOrder(err, o)
		}
	}
	return nil
}

func errSubmitOrder(err error, o *types.Order) error {
	return fmt.Errorf("error submitting order [%v]: %v", o, err)
}
