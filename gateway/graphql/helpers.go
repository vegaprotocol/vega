package gql

import (
	"fmt"
	"strconv"

	types "code.vegaprotocol.io/vega/proto"
)

func safeStringUint64(input string) (uint64, error) {
	if i, err := strconv.ParseUint(input, 10, 64); err == nil {
		return i, nil
	}
	// A conversion error occurred, return the error
	return 0, fmt.Errorf("invalid input string for uint64 conversion %s", input)
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
		err := fmt.Errorf("invalid interval when subscribing to candles, falling back to default: I15M, (%v)", interval)

		return types.Interval_I15M, err
	}
}

func parseOrderTimeInForce(timeInForce OrderTimeInForce) (types.Order_TimeInForce, error) {
	switch timeInForce {
	case OrderTimeInForceGtc:
		return types.Order_GTC, nil
	case OrderTimeInForceGtt:
		return types.Order_GTT, nil
	case OrderTimeInForceIoc:
		return types.Order_IOC, nil
	case OrderTimeInForceFok:
		return types.Order_FOK, nil
	default:
		return types.Order_GTC, fmt.Errorf("unknown type: %s", timeInForce.String())
	}
}

func parseOrderType(ty OrderType) (types.Order_Type, error) {
	switch ty {
	case OrderTypeLimit:
		return types.Order_LIMIT, nil
	case OrderTypeMarket:
		return types.Order_MARKET, nil
	default:
		// handle types.Order_NETWORK as an error here, as we do not expected
		// it to be set by through the API, only by the core internally
		return 0, fmt.Errorf("unknown type: %s", ty.String())
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
	case OrderStatusRejected:
		return types.Order_Rejected, nil
	default:
		return types.Order_Active, fmt.Errorf("unknown status: %s", orderStatus.String())
	}
}

func parseSide(side *Side) (types.Side, error) {
	switch *side {
	case SideBuy:
		return types.Side_Buy, nil
	case SideSell:
		return types.Side_Sell, nil
	default:
		return types.Side_Buy, fmt.Errorf("unknown side: %s", side.String())
	}
}
