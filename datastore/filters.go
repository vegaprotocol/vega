package datastore

import "vega/common"

// applyOrderFilters takes an incoming set of OrderQueryFilters and applies them
// to the specified order. Internally the OrderQueryFilters will set operator e.g. AND/OR
func applyOrderFilters(order Order, filters *common.OrderQueryFilters) bool {
	ok := true
	count := 0

	if filters.IdFilter != nil {
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

	if filters.Operator == common.QueryFilterOperatorAnd {
		// If we AND all the filters the counts should match
		// and if they do we have the exact match
		return count == filters.Count()
	} else {
		// We are in an OR operation so if any of the filters
		// have matched we can return true, false otherwise
		return ok
	}
}

func applyTradeFilters(trade Trade, filters *common.TradeQueryFilters) bool {
	ok := true
	count := 0

	if filters.IdFilter != nil {
		ok = filters.IdFilter.ApplyFilters(trade.Id)
		if ok {
			count++
		}
	}
	if filters.MarketFilter != nil {
		ok = filters.MarketFilter.ApplyFilters(trade.Market)
		if ok {
			count++
		}
	}
	if filters.PriceFilter != nil {
		ok = filters.PriceFilter.ApplyFilters(trade.Price)
		if ok {
			count++
		}
	}
	if filters.SizeFilter != nil {
		ok = filters.SizeFilter.ApplyFilters(trade.Size)
		if ok {
			count++
		}
	}
	if filters.BuyerFilter != nil {
		ok = filters.BuyerFilter.ApplyFilters(trade.Buyer)
		if ok {
			count++
		}
	}
	if filters.SellerFilter != nil {
		ok = filters.SellerFilter.ApplyFilters(trade.Seller)
		if ok {
			count++
		}
	}
	if filters.AggressorFilter != nil {
		ok = filters.AggressorFilter.ApplyFilters(trade.Aggressor)
		if ok {
			count++
		}
	}
	if filters.TimestampFilter != nil {
		ok = filters.TimestampFilter.ApplyFilters(trade.Timestamp)
		if ok {
			count++
		}
	}

	if filters.Operator == common.QueryFilterOperatorAnd {
		// If we AND all the filters the counts should match
		// and if they do we have the exact match
		return count == filters.Count()
	} else {
		// We are in an OR operation so if any of the filters
		// have matched we can return true, false otherwise
		return ok
	}
}

