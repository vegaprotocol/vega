package gql

import (
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/internal/filtering"
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

func buildOrderQueryFilters(where *OrderFilter, skip *int, first *int, last *int) (queryFilters *filtering.OrderQueryFilters, err error) {
	if queryFilters == nil {
		queryFilters = &filtering.OrderQueryFilters{}
	}
	if where != nil {

		// AND default
		queryFilters.Operator = filtering.QueryFilterOperatorAnd
		if where.OR != nil {
			if where.AND != nil {
				return nil, errors.New("combination of operators is not currently supported")
			}
			for _, filter := range where.OR {
				_, err := ParseOrderFilter(&filter, queryFilters)
				if err != nil {
					return nil, err
				}
			}
			// If OR specified switch operator to OR inc outer filters
			queryFilters.Operator = filtering.QueryFilterOperatorOr
		} else if where.AND != nil {
			for _, filter := range where.AND {
				_, err := ParseOrderFilter(&filter, queryFilters)
				if err != nil {
					return nil, err
				}
			}
		}
		// Always parse outer filters
		_, err = ParseOrderFilter(where, queryFilters)
		if err != nil {
			return nil, err
		}
	}

	// Parse pagination params (if set)
	if last != nil {
		l := uint64(*last)
		queryFilters.Last = &l
	}
	if first != nil {
		if last != nil {
			return nil, errors.New("first and last cannot both be specified in query")
		}
		f := uint64(*first)
		queryFilters.First = &f
	}
	if skip != nil {
		s := uint64(*skip)
		queryFilters.Skip = &s
	}

	return queryFilters, nil
}

func buildTradeQueryFilters(where *TradeFilter, skip *int, first *int, last *int) (queryFilters *filtering.TradeQueryFilters, err error) {
	if queryFilters == nil {
		queryFilters = &filtering.TradeQueryFilters{}
	}

	// Parse 'where' and build query filters that will be used internally (if set)
	if where != nil {

		// AND default
		queryFilters.Operator = filtering.QueryFilterOperatorAnd
		if where.OR != nil {
			if where.AND != nil {
				return nil, errors.New("combination of operators is not currently supported")
			}
			for _, filter := range where.OR {
				_, err := ParseTradeFilter(&filter, queryFilters)
				if err != nil {
					return nil, err
				}
			}
			// If OR specified switch operator to OR inc outer filters
			queryFilters.Operator = filtering.QueryFilterOperatorOr
		} else if where.AND != nil {
			for _, filter := range where.AND {
				_, err := ParseTradeFilter(&filter, queryFilters)
				if err != nil {
					return nil, err
				}
			}
		}
		// Always parse outer filters
		_, err = ParseTradeFilter(where, queryFilters)
		if err != nil {
			return nil, err
		}
	}

	// Parse pagination params (if set)
	if last != nil {
		l := uint64(*last)
		queryFilters.Last = &l
	}
	if first != nil {
		if last != nil {
			return nil, errors.New("first and last cannot both be specified in query")
		}
		f := uint64(*first)
		queryFilters.First = &f
	}
	if skip != nil {
		s := uint64(*skip)
		queryFilters.Skip = &s
	}

	return queryFilters, nil
}
