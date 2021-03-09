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

func panicW(err error) {
	if err != nil {
		panic(err)
	}
}

func TradersPlaceFollowingOrders(
	exec *execution.Engine,
	orders *gherkin.DataTable,
) error {
	for _, row := range TableWrapper(*orders).Parse() {
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
			MarketId:    row.Str("id"), // this is actually the market id
			PartyId:     row.Str("trader"),
			Side:        side,
			Price:       price,
			Size:        volume,
			Remaining:   volume,
			ExpiresAt:   expiresAt,
			Type:        oty,
			TimeInForce: tif,
			CreatedAt:   time.Now().UnixNano(),
		}
		result, err := exec.SubmitOrder(context.Background(), &order)
		if err != nil {
			return fmt.Errorf("unable to place order, err=%v (trader=%v)", err, row.Str("trader"))
		}

		resultinTrades, err := row.U64("resulting trades")
		panicW(err)

		if uint64(len(result.Trades)) != resultinTrades {
			return fmt.Errorf("expected %d trades, instead saw %d (%#v)", resultinTrades, len(result.Trades), *result)
		}
	}
	return nil
}
