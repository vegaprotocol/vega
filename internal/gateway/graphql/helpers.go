package gql

import (
	"fmt"
	"strconv"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

func safeStringUint64(input string) (uint64, error) {
	if i, err := strconv.ParseUint(input, 10, 64); err == nil {
		return i, nil
	}
	// A conversion error occurred, return the error
	return 0, errors.New(fmt.Sprintf("Invalid input string for uint64 conversion %s", input))
}

func convertInterval(interval Interval) (types.Interval, error) {
	switch interval {
	case IntervalI15m:
		return types.Interval_I15M, nil
	case IntervalI1d:
		return types.Interval_I1D, nil
	case IntervalI1h:
		return types.Interval_I1H, nil
	case IntervalI1m:
		return types.Interval_I1M, nil
	case IntervalI5m:
		return types.Interval_I5M, nil
	case IntervalI6h:
		return types.Interval_I6H, nil
	default:
		err := fmt.Errorf("Invalid interval when subscribing to candles, falling back to default: I15M, (%v)", interval)

		return types.Interval_I15M, err
	}
}

func parseOrderType(orderType *OrderType) (types.Order_Type, error) {
	switch *orderType {
	case OrderTypeGtc:
		return types.Order_GTC, nil
	case OrderTypeGtt:
		return types.Order_GTT, nil
	case OrderTypeEne:
		return types.Order_ENE, nil
	case OrderTypeFok:
		return types.Order_FOK, nil
	default:
		return types.Order_GTC, errors.New(fmt.Sprintf("unknown type: %s", orderType.String()))
	}
}

func parseOrderStatus(orderStatus *OrderStatus) (types.Order_Status, error) {
	switch *orderStatus {
	case OrderStatusActive:
		return types.Order_Active, nil
	case OrderStatusExpired:
		return types.Order_Expired, nil
	case OrderStatusCancelled:
		return types.Order_Cancelled, nil
	case OrderStatusFilled:
		return types.Order_Filled, nil
	default:
		return types.Order_Active, errors.New(fmt.Sprintf("unknown status: %s", orderStatus.String()))
	}
}

func parseSide(side *Side) (types.Side, error) {
	switch *side {
	case SideBuy:
		return types.Side_Buy, nil
	case SideSell:
		return types.Side_Sell, nil
	default:
		return types.Side_Buy, errors.New(fmt.Sprintf("unknown side: %s", side.String()))
	}
}
