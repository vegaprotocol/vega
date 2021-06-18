package steps

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/integration/helpers"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
)

func TradersPlaceTheFollowingOrders(
	exec *execution.Engine,
	errorHandler *helpers.ErrorHandler,
	table *gherkin.DataTable,
) error {
	for _, row := range TableWrapper(*table).Parse() {
		trader := row.MustStr("trader")
		marketID := row.MustStr("market id")
		side := row.MustSide("side")
		volume := row.MustU64("volume")
		price := row.MustU64("price")
		oty := row.MustOrderType("type")
		tif := row.MustTIF("tif")
		reference := row.Str("reference")

		var resultingTrades int64 = -1
		if row.Str("resulting trades") != "" {
			resultingTrades = row.I64("resulting trades")
			if resultingTrades > 0 && row.Str("error") != "" {
				panic("you can't expect resulting trades and an error at the same time")
			}
		}

		var expiresAt int64
		if oty != types.Order_TYPE_MARKET {
			now := time.Now()
			if tif == types.Order_TIME_IN_FORCE_GTT {
				expiresAt = now.Add(row.MustDurationSec("expires in")).Local().UnixNano()
			} else {
				expiresAt = now.Add(24 * time.Hour).UnixNano()
			}
		}

		orderSubmission := commandspb.OrderSubmission{
			MarketId:    marketID,
			Side:        side,
			Price:       price,
			Size:        volume,
			ExpiresAt:   expiresAt,
			Type:        oty,
			TimeInForce: tif,
			Reference:   reference,
		}

		resp, err := exec.SubmitOrder(context.Background(), &orderSubmission, trader)
		if err != nil {
			errMsg := row.Str("error")
			if err.Error() != errMsg {
				return formatDiff(fmt.Sprintf("the order \"%v\" is failing as expected but not with the expected error message", reference),
					map[string]string{
						"error": errMsg,
					},
					map[string]string{
						"error": err.Error(),
					},
				)
			}
			return nil
		}

		if resultingTrades != -1 && len(resp.Trades) != int(resultingTrades) {
			return formatDiff(fmt.Sprintf("the resulting trades didn't match the expectation for order \"%v\"", reference),
				map[string]string{
					"total": fmt.Sprintf("%v", resultingTrades),
				},
				map[string]string{
					"total": fmt.Sprintf("%v", len(resp.Trades)),
				},
			)
		}
	}
	return nil
}
