package gql

import (
	"strconv"
	"fmt"
	"github.com/pkg/errors"
	"vega/msg"
	"vega/common"
	"vega/log"
)

func SafeStringUint64(input string) (uint64, error) {
	if i, err := strconv.ParseUint(input, 10, 64); err == nil {
		fmt.Printf("i=%d, type: %T\n", i, i)
		return i, nil
	}
	// A conversion error occurred, return the error
	return 0, errors.New(fmt.Sprintf("Invalid input string for uint64 conversion %s", input))
}

func ParseOrderType(orderType *OrderType) (msg.Order_Type, error) {
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

func ParseOrderStatus(orderStatus *OrderStatus) (msg.Order_Status, error) {
	switch *orderStatus {
	case OrderStatusActive:
		return msg.Order_Active, nil
	case OrderStatusExpired:
		return msg.Order_Expired, nil
	case OrderStatusCancelled:
		return msg.Order_Cancelled, nil
	default:
		return msg.Order_Active, errors.New(fmt.Sprintf("unknown status: %s", orderStatus.String()))
	}
}

func ParseSide(side *Side) (msg.Side, error) {
	switch *side {
		case SideBuy:
			return msg.Side_Buy, nil
		case SideSell:
			return msg.Side_Sell, nil
		default:
			return msg.Side_Buy, errors.New(fmt.Sprintf("unknown side: %s", side.String()))
	}
}


func buildOrderQueryFilters(where *OrderFilter, skip *int, first *int, last *int) (queryFilters *common.OrderQueryFilters, err error) {
	if queryFilters == nil {
		queryFilters = &common.OrderQueryFilters{}
	}
	if where != nil {
		log.Debugf("OrderFilters: %+v", where)

		// OR default
		queryFilters.Operator = common.QueryFilterOperatorOr
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
		} else if where.AND != nil {
			for _, filter := range where.AND {
				_, err := ParseOrderFilter(&filter, queryFilters)
				if err != nil {
					return nil, err
				}
			}
			// If AND specified switch operator to AND inc outer filters
			queryFilters.Operator = common.QueryFilterOperatorAnd
		}
		// Always parse outer filters
		_, err := ParseOrderFilter(where, queryFilters)
		if err != nil {
			return nil, err
		}
		if last != nil {
			*queryFilters.Last = uint64(*last)
		}
		// todo(cdm): first (asc), skip (directional ffwd)
	}

	return queryFilters, nil
}


func buildTradeQueryFilters(where *TradeFilter, skip *int, first *int, last *int) (queryFilters *common.TradeQueryFilters, err error) {
	if queryFilters == nil {
		queryFilters = &common.TradeQueryFilters{}
	}
	if where != nil {
		log.Debugf("TradeFilters: %+v", where)

		// OR default
		queryFilters.Operator = common.QueryFilterOperatorOr
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
		} else if where.AND != nil {
			for _, filter := range where.AND {
				_, err := ParseTradeFilter(&filter, queryFilters)
				if err != nil {
					return nil, err
				}
			}
			// If AND specified switch operator to AND inc outer filters
			queryFilters.Operator = common.QueryFilterOperatorAnd
		}
		// Always parse outer filters
		_, err := ParseTradeFilter(where, queryFilters)
		if err != nil {
			return nil, err
		}
		if last != nil {
			*queryFilters.Last = uint64(*last)
		}
		// todo(cdm): first (asc), skip (directional ffwd)
	}

	return queryFilters, nil
}


