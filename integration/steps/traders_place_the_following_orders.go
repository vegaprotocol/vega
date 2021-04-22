package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/cucumber/godog/gherkin"
	uuid "github.com/satori/go.uuid"

	"code.vegaprotocol.io/vega/integration/helpers"

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

		order := types.Order{
			Status:      types.Order_STATUS_ACTIVE,
			Id:          uuid.NewV4().String(),
			MarketId:    marketID,
			PartyId:     trader,
			Side:        side,
			Price:       price,
			Size:        volume,
			Remaining:   volume,
			ExpiresAt:   expiresAt,
			Type:        oty,
			TimeInForce: tif,
			CreatedAt:   time.Now().UnixNano(),
			Reference:   reference,
		}

		resp, err := exec.SubmitOrder(context.Background(), &order)
		if err != nil {
			errorHandler.HandleError(SubmitOrderError{
				reference: reference,
				request:   order,
				Err:       err,
			})
			return fmt.Errorf("could not submit order %w", err)
		}

		if resultingTrades != -1 && len(resp.Trades) != int(resultingTrades) {
			errorHandler.HandleError(SubmitOrderError{
				reference: reference,
				request:   order,
				Err:       fmt.Errorf("expected %d trades executed, but got %d confirmations", resultingTrades, len(resp.Trades)),
			})
		}
	}
	return nil
}
