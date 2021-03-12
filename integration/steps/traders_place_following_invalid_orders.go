package steps

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/cucumber/godog/gherkin"
	uuid "github.com/satori/go.uuid"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
)

func TradersPlaceFollowingInvalidOrders(
	exec *execution.Engine,
	table *gherkin.DataTable,
) error {
	for _, row := range TableWrapper(*table).Parse() {
		oty := row.OrderType("type")
		tif := row.TIF("tif")
		side := row.Side("side")
		price := row.U64("price")
		volume := row.U64("volume")
		trader := row.Str("trader")
		reference := strconv.FormatInt(time.Now().UnixNano(), 10)
		expectedError := row.Str("error")

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

		_, err := exec.SubmitOrder(context.Background(), &order)

		if err == nil {
			return errUnexpectedSuccessfulOrder(expectedError, err)
		}
		if err.Error() != expectedError {
			return errMismatchedErrorMessage(expectedError, err)
		}
	}
	return nil
}

func errMismatchedErrorMessage(expectedError string, err error) error {
	return fmt.Errorf("expected error (%v) but got (%v)", expectedError, err)
}

func errUnexpectedSuccessfulOrder(expectedError string, err error) error {
	return fmt.Errorf("expected error (%v) but got (%v)", expectedError, err)
}
