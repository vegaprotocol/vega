package steps

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/cucumber/godog/gherkin"
	uuid "github.com/satori/go.uuid"
)

func TradersPlaceFollowingOrders(
	exec *execution.Engine,
	orders *gherkin.DataTable,
) error {
	for _, row := range TableWrapper(*orders).Parse() {
		oty := row.OrderType("type")
		tif := row.TIF("tif")
		side := row.Side("side")
		price := row.U64("price")
		volume := row.U64("volume")
		trader := row.Str("trader")
		reference := strconv.FormatInt(time.Now().UnixNano(), 10)

		var expiresAt int64
		if oty != types.Order_TYPE_MARKET {
			expiresAt = time.Now().Add(24 * time.Hour).UnixNano()
		}

		order := types.Order{
			Status:      types.Order_STATUS_ACTIVE,
			Id:          uuid.NewV4().String(),
			MarketId:    row.Str("market id"),
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
		result, err := exec.SubmitOrder(context.Background(), &order)
		if err != nil {
			return errUnableToPlaceOrder(trader, reference, err)
		}

		resultingTrades := row.U64("resulting trades")

		if uint64(len(result.Trades)) != resultingTrades {
			return fmt.Errorf("expected %d trades, instead saw %d (%#v)", resultingTrades, len(result.Trades), *result)
		}
	}
	return nil
}
