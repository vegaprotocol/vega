package steps

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
	uuid "github.com/satori/go.uuid"
)

func TradersPlaceFollowingOrdersWithReferences(
	exec *execution.Engine,
	table *gherkin.DataTable,
) error {
	for _, row := range TableWrapper(*table).Parse() {
		oty, err := row.OrderType("type")
		panicW(err)
		tif, err := row.TIF("tif")
		panicW(err)
		side, err := row.Side("side")
		panicW(err)
		price, err := row.U64("price")
		panicW(err)
		volume, err := row.U64("volume")
		panicW(err)

		var expiresAt int64
		if oty != types.Order_TYPE_MARKET {
			expiresAt = time.Now().Add(24 * time.Hour).UnixNano()
		}

		order := types.Order{
			Status:      types.Order_STATUS_ACTIVE,
			Id:          uuid.NewV4().String(),
			MarketId:    row.Str("market id"),
			PartyId:     row.Str("trader"),
			Side:        side,
			Price:       price,
			Size:        volume,
			Remaining:   volume,
			ExpiresAt:   expiresAt,
			Type:        oty,
			TimeInForce: tif,
			CreatedAt:   time.Now().UnixNano(),
			Reference:   row.Str("reference"),
		}
		result, err := exec.SubmitOrder(context.Background(), &order)
		if err != nil {
			return fmt.Errorf("err(%v), trader(%v), ref(%v)",
				err, order.PartyId, order.Reference)
		}
		resultingTrades, err := row.U64("resulting trades")
		panicW(err)
		if len(result.Trades) != int(resultingTrades) {
			return fmt.Errorf(
				"expected %d trades, instead saw %d (%#v)",
				resultingTrades, len(result.Trades), *result,
			)
		}
	}
	return nil
}
