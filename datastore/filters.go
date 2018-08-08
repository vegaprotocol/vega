package datastore

import("vega/common"
	"fmt"
)

// GetParamsLimitDefault should be used if no limit is specified
// when working with the GetParams struct.
const GetParamsLimitDefault = uint64(1844674407370955161)

//type GetOrderParams struct {
//	common.QueryFilterPaginated
//
//	Limit           uint64
//
//	MarketFilter    *common.QueryFilter
//	PartyFilter     *common.QueryFilter
//	SideFilter      *common.QueryFilter
//	PriceFilter     *common.QueryFilter
//	SizeFilter      *common.QueryFilter
//	RemainingFilter *common.QueryFilter
//	TypeFilter      *common.QueryFilter
//	TimestampFilter *common.QueryFilter
//	RiskFactor      *common.QueryFilter
//	StatusFilter    *common.QueryFilter
//}

type GetTradeParams struct {
	common.QueryFilterPaginated

	Limit           uint64
	
	MarketFilter    *common.QueryFilter
	PriceFilter     *common.QueryFilter
	SizeFilter      *common.QueryFilter
	BuyerFilter     *common.QueryFilter
	SellerFilter    *common.QueryFilter
	AggressorFilter *common.QueryFilter
	TimestampFilter *common.QueryFilter
}

func applyOrderFilter(order Order, filters *common.OrderQueryFilters, op common.QueryFilterOperation) bool {
	ok := true
	count := 0

	if filters.IdFilter != nil {

		fmt.Println(fmt.Sprintf("ID in filters datastore: %+v %+v", filters.IdFilter.Eq, order.Id))

		ok = filters.IdFilter.ApplyFilters(order.Id)
		if ok {
			count++
		}
	}

	if filters.MarketFilter != nil {
		ok = filters.MarketFilter.ApplyFilters(order.Market)
		if ok {
			count++
		}
	}

	if filters.PartyFilter != nil {
		ok = filters.PartyFilter.ApplyFilters(order.Party)
		if ok {
			count++
		}
	}

	if filters.SideFilter != nil {
		ok = filters.SideFilter.ApplyFilters(order.Side)
		if ok {
			count++
		}
	}

	if filters.PriceFilter != nil {
		ok = filters.PriceFilter.ApplyFilters(order.Price)
		if ok {
			count++
		}
	}

	if filters.SizeFilter != nil {
		ok = filters.SizeFilter.ApplyFilters(order.Size)
		if ok {
			count++
		}
	}

	if filters.RemainingFilter != nil {
		
		fmt.Println("Remaining in filters datastore: ", filters.RemainingFilter.Eq, order.Remaining)

		ok = filters.RemainingFilter.ApplyFilters(order.Remaining)
		if ok {
			count++
		}
	}

	if filters.TypeFilter != nil {
		ok = filters.TypeFilter.ApplyFilters(order.Type)
		if ok {
			count++
		}
	}

	if filters.TimestampFilter != nil {
		ok = filters.TimestampFilter.ApplyFilters(order.Timestamp)
		if ok {
			count++
		}
	}
	
	if filters.StatusFilter != nil {
		ok = filters.StatusFilter.ApplyFilters(order.Status)
		if ok {
			count++
		}
	}

	if op == common.QueryFilterOperationAnd {
		// If we AND all the filters the counts should match
		// and if they do we have the exact match
		return count == filters.Count()
	} else {
		// We are in an OR operation so if any of the filters
		// have matched we can return true, false otherwise
		return ok
	}

}

func applyTradeFilter(trade Trade, params GetTradeParams) bool {
	var ok = true

	if params.MarketFilter != nil {
		ok = params.MarketFilter.ApplyFilters(trade.Market)
	}

	if params.PriceFilter != nil {
		ok = params.PriceFilter.ApplyFilters(trade.Price)
	}

	if params.SizeFilter != nil {
		ok = params.SizeFilter.ApplyFilters(trade.Size)
	}

	if params.BuyerFilter != nil {
		ok = params.BuyerFilter.ApplyFilters(trade.Buyer)
	}

	if params.SellerFilter != nil {
		ok = params.SellerFilter.ApplyFilters(trade.Seller)
	}

	if params.AggressorFilter != nil {
		ok = params.AggressorFilter.ApplyFilters(trade.Aggressor)
	}

	if params.TimestampFilter != nil {
		ok = params.TimestampFilter.ApplyFilters(trade.Timestamp)
	}

	return ok
}

