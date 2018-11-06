package gql

import (
	"strconv"
	"fmt"
	"github.com/pkg/errors"
	"vega/msg"
	"vega/filters"
)

func safeStringUint64(input string) (uint64, error) {
	if i, err := strconv.ParseUint(input, 10, 64); err == nil {
		return i, nil
	}
	// A conversion error occurred, return the error
	return 0, errors.New(fmt.Sprintf("Invalid input string for uint64 conversion %s", input))
}

func parseOrderType(orderType *OrderType) (msg.Order_Type, error) {
	switch *orderType {
		case OrderTypeGtc:
			return msg.Order_GTC, nil
		case OrderTypeGtt:
			return msg.Order_GTT, nil
		case OrderTypeEne:
			return msg.Order_ENE, nil
		case OrderTypeFok:
			return msg.Order_FOK, nil
		default:
			return msg.Order_GTC, errors.New(fmt.Sprintf("unknown type: %s", orderType.String()))
	}
}

func parseOrderStatus(orderStatus *OrderStatus) (msg.Order_Status, error) {
	switch *orderStatus {
		case OrderStatusActive:
			return msg.Order_Active, nil
		case OrderStatusExpired:
			return msg.Order_Expired, nil
		case OrderStatusCancelled:
			return msg.Order_Cancelled, nil
		case OrderStatusFilled:
			return msg.Order_Filled, nil
		default:
			return msg.Order_Active, errors.New(fmt.Sprintf("unknown status: %s", orderStatus.String()))
	}
}

func parseSide(side *Side) (msg.Side, error) {
	switch *side {
		case SideBuy:
			return msg.Side_Buy, nil
		case SideSell:
			return msg.Side_Sell, nil
		default:
			return msg.Side_Buy, errors.New(fmt.Sprintf("unknown side: %s", side.String()))
	}
}


func buildOrderQueryFilters(where *OrderFilter, skip *int, first *int, last *int) (queryFilters *filters.OrderQueryFilters, err error) {
	if queryFilters == nil {
		queryFilters = &filters.OrderQueryFilters{}
	}
	if where != nil {
		//log.Debugf("OrderFilters: %+v", where)

		// AND default
		queryFilters.Operator = filters.QueryFilterOperatorAnd
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
			queryFilters.Operator = filters.QueryFilterOperatorOr
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


func buildTradeQueryFilters(where *TradeFilter, skip *int, first *int, last *int) (queryFilters *filters.TradeQueryFilters, err error) {
	if queryFilters == nil {
		queryFilters = &filters.TradeQueryFilters{}
	}

	// Parse 'where' and build query filters that will be used internally (if set)
	if where != nil {
		//log.Debugf("TradeFilters: %+v", where)

		// AND default
		queryFilters.Operator = filters.QueryFilterOperatorAnd
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
			queryFilters.Operator = filters.QueryFilterOperatorOr
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
