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
		}

		var expiresAt int64
		if oty != types.Order_TYPE_MARKET {
			expiresAt = time.Now().Add(24 * time.Hour).UnixNano()
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
			errorHandler.HandleError(SubmitOrderError{
				reference: reference,
				request:   orderSubmission,
				Err:       err,
			})
			return nil
		}

		if resultingTrades != -1 && len(resp.Trades) != int(resultingTrades) {
			errorHandler.HandleError(SubmitOrderError{
				reference: reference,
				request:   orderSubmission,
				Err:       fmt.Errorf("expected %d trades executed, but got %d confirmations", resultingTrades, len(resp.Trades)),
			})
		}
	}
	return nil
}
