package gql

import (
	"fmt"
	"strconv"

	"github.com/vektah/gqlparser/gqlerror"
	"google.golang.org/grpc/status"

	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

func safeStringUint64(input string) (uint64, error) {
	if i, err := strconv.ParseUint(input, 10, 64); err == nil {
		return i, nil
	}
	// A conversion error occurred, return the error
	return 0, fmt.Errorf("invalid input string for uint64 conversion %s", input)
}

// convertInterval converts a GraphQL enum to a Proto enum
func convertInterval(interval Interval) (types.Interval, error) {
	switch interval {
	case IntervalI15m:
		return types.Interval_INTERVAL_I15M, nil
	case IntervalI1d:
		return types.Interval_INTERVAL_I1D, nil
	case IntervalI1h:
		return types.Interval_INTERVAL_I1H, nil
	case IntervalI1m:
		return types.Interval_INTERVAL_I1M, nil
	case IntervalI5m:
		return types.Interval_INTERVAL_I5M, nil
	case IntervalI6h:
		return types.Interval_INTERVAL_I6H, nil
	default:
		err := fmt.Errorf("failed to convert Interval from GraphQL to Proto: %v", interval)
		return types.Interval_INTERVAL_UNSPECIFIED, err
	}
}

// unconvertInterval converts a Proto enum to a Proto enum
func unconvertInterval(interval types.Interval) (Interval, error) {
	switch interval {
	case types.Interval_INTERVAL_I15M:
		return IntervalI15m, nil
	case types.Interval_INTERVAL_I1D:
		return IntervalI1d, nil
	case types.Interval_INTERVAL_I1H:
		return IntervalI1h, nil
	case types.Interval_INTERVAL_I1M:
		return IntervalI1m, nil
	case types.Interval_INTERVAL_I5M:
		return IntervalI5m, nil
	case types.Interval_INTERVAL_I6H:
		return IntervalI6h, nil
	default:
		err := fmt.Errorf("failed to convert Interval from Proto to GraphQL: %v", interval)
		return IntervalI15m, err
	}
}

func parseOrderTimeInForce(timeInForce OrderTimeInForce) (types.Order_TimeInForce, error) {
	switch timeInForce {
	case OrderTimeInForceGtc:
		return types.Order_TIF_GTC, nil
	case OrderTimeInForceGtt:
		return types.Order_TIF_GTT, nil
	case OrderTimeInForceIoc:
		return types.Order_TIF_IOC, nil
	case OrderTimeInForceFok:
		return types.Order_TIF_FOK, nil
	default:
		return types.Order_TIF_UNSPECIFIED, fmt.Errorf("unknown type: %s", timeInForce.String())
	}
}

func parseOrderType(ty OrderType) (types.Order_Type, error) {
	switch ty {
	case OrderTypeLimit:
		return types.Order_TYPE_LIMIT, nil
	case OrderTypeMarket:
		return types.Order_TYPE_MARKET, nil
	default:
		// handle types.Order_TYPE_NETWORK as an error here, as we do not expected
		// it to be set by through the API, only by the core internally
		return types.Order_TYPE_UNSPECIFIED, fmt.Errorf("unknown type: %s", ty.String())
	}
}

// convertOrderStatus converts a GraphQL enum to a Proto enum
func convertOrderStatus(orderStatus *OrderStatus) (types.Order_Status, error) {
	switch *orderStatus {
	case OrderStatusActive:
		return types.Order_STATUS_ACTIVE, nil
	case OrderStatusExpired:
		return types.Order_STATUS_EXPIRED, nil
	case OrderStatusCancelled:
		return types.Order_STATUS_CANCELLED, nil
	case OrderStatusStopped:
		return types.Order_STATUS_STOPPED, nil
	case OrderStatusFilled:
		return types.Order_STATUS_FILLED, nil
	case OrderStatusRejected:
		return types.Order_STATUS_REJECTED, nil
	case OrderStatusPartiallyFilled:
		return types.Order_STATUS_PARTIALLY_FILLED, nil
	default:
		err := fmt.Errorf("failed to convert OrderStatus from GraphQL to Proto: %v", orderStatus)
		return types.Order_STATUS_INVALID, err
	}
}

// unconvertOrderStatus converts a Proto enum to a GraphQL enum
func unconvertOrderStatus(orderStatus types.Order_Status) (OrderStatus, error) {
	switch orderStatus {
	case types.Order_STATUS_ACTIVE:
		return OrderStatusActive, nil
	case types.Order_STATUS_EXPIRED:
		return OrderStatusExpired, nil
	case types.Order_STATUS_CANCELLED:
		return OrderStatusCancelled, nil
	case types.Order_STATUS_STOPPED:
		return OrderStatusStopped, nil
	case types.Order_STATUS_FILLED:
		return OrderStatusFilled, nil
	case types.Order_STATUS_REJECTED:
		return OrderStatusRejected, nil
	case types.Order_STATUS_PARTIALLY_FILLED:
		return OrderStatusPartiallyFilled, nil
	default:
		err := fmt.Errorf("failed to convert OrderStatus from Proto to GraphQL: %v", orderStatus)
		return OrderStatusActive, err
	}
}

// convertSide converts a GraphQL enum to a Proto enum
func convertSide(side *Side) (types.Side, error) {
	switch *side {
	case SideBuy:
		return types.Side_SIDE_BUY, nil
	case SideSell:
		return types.Side_SIDE_SELL, nil
	default:
		err := fmt.Errorf("failed to convert Side from GraphQL to Proto: %v", side)
		return types.Side_SIDE_UNSPECIFIED, err
	}
}

// unconvertSide converts a Proto enum to a GraphQL enum
func unconvertSide(side types.Side) (Side, error) {
	switch side {
	case types.Side_SIDE_BUY:
		return SideBuy, nil
	case types.Side_SIDE_SELL:
		return SideSell, nil
	default:
		err := fmt.Errorf("failed to convert Side from Proto to GraphQL: %v", side)
		return SideBuy, err
	}
}

// customErrorFromStatus provides a richer error experience from grpc ErrorDetails
// which is provided by the Vega grpc API. This helper takes in the error provided
// by a grpc client and either returns a custom graphql error or the raw error string.
func customErrorFromStatus(err error) error {
	st, ok := status.FromError(err)
	if ok {
		customCode := ""
		customDetail := ""
		customInner := ""
		customMessage := st.Message()
		errorDetails := st.Details()
		if errorDetails != nil {
			for _, s := range errorDetails {
				det := s.(*types.ErrorDetail)
				customDetail = det.Message
				customCode = fmt.Sprintf("%d", det.Code)
				customInner = det.Inner
				break
			}
		}
		return &gqlerror.Error{
			Message: customMessage,
			Extensions: map[string]interface{}{
				"detail": customDetail,
				"code":   customCode,
				"inner":  customInner,
			},
		}
	}
	return err
}

func secondsTSToDatetime(timestampInSeconds int64) string {
	return vegatime.Format(vegatime.Unix(timestampInSeconds, 0))
}

func nanoTSToDatetime(timestampInNanoSeconds int64) string {
	return vegatime.Format(vegatime.UnixNano(timestampInNanoSeconds))
}

func datetimeToSecondsTS(timestamp string) (int64, error) {
	converted, err := vegatime.Parse(timestamp)
	if err != nil {
		return 0, err
	}
	return converted.UTC().Unix(), nil
}

func removePointers(input []*string) []string {
	result := make([]string, 0, len(input))
	for _, sPtr := range input {
		if sPtr != nil {
			result = append(result, *sPtr)
		}
	}
	return result
}

func convertVersion(version *int) (uint64, error) {
	const defaultValue = 0

	if version != nil {
		if *version >= 0 {
			return uint64(*version), nil
		}
		return defaultValue, fmt.Errorf("invalid version value %d", *version)
	}
	return defaultValue, nil
}
