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
		err := fmt.Errorf("invalid interval when subscribing to candles, falling back to default: I15M, (%v)", interval)

		return types.Interval_INTERVAL_I15M, err
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
		return types.Order_STATUS_ACTIVE, nil
	case OrderStatusExpired:
		return types.Order_STATUS_EXPIRED, nil
	case OrderStatusCancelled:
		return types.Order_STATUS_CANCELLED, nil
	case OrderStatusFilled:
		return types.Order_STATUS_FILLED, nil
	case OrderStatusRejected:
		return types.Order_STATUS_REJECTED, nil
	default:
		return types.Order_STATUS_ACTIVE, fmt.Errorf("unknown status: %s", orderStatus.String())
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
