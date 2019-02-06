package storage

import (
	"vega/internal/filtering"
	"vega/msg"
)

func applyTradeFilters(trade *msg.Trade, queryFilters *filtering.TradeQueryFilters) bool {
	ok := true
	count := 0

	if queryFilters.IdFilter != nil {
		ok = queryFilters.IdFilter.ApplyFilters(trade.Id)
		if ok {
			count++
		}
	}
	if queryFilters.MarketFilter != nil {
		ok = queryFilters.MarketFilter.ApplyFilters(trade.Market)
		if ok {
			count++
		}
	}
	if queryFilters.PriceFilter != nil {
		ok = queryFilters.PriceFilter.ApplyFilters(trade.Price)
		if ok {
			count++
		}
	}
	if queryFilters.SizeFilter != nil {
		ok = queryFilters.SizeFilter.ApplyFilters(trade.Size)
		if ok {
			count++
		}
	}
	if queryFilters.BuyerFilter != nil {
		ok = queryFilters.BuyerFilter.ApplyFilters(trade.Buyer)
		if ok {
			count++
		}
	}
	if queryFilters.SellerFilter != nil {
		ok = queryFilters.SellerFilter.ApplyFilters(trade.Seller)
		if ok {
			count++
		}
	}
	if queryFilters.AggressorFilter != nil {
		ok = queryFilters.AggressorFilter.ApplyFilters(trade.Aggressor)
		if ok {
			count++
		}
	}
	if queryFilters.TimestampFilter != nil {
		ok = queryFilters.TimestampFilter.ApplyFilters(trade.Timestamp)
		if ok {
			count++
		}
	}

	if queryFilters.Operator == filtering.QueryFilterOperatorAnd {
		// If we AND all the queryFilters the counts should match
		// and if they do we have the exact match
		return count == queryFilters.Count()
	} else {
		// We are in an OR operation so if any of the queryFilters
		// have matched we can return true, false otherwise
		return ok
	}
}

func applyOrderFilters(order *msg.Order, queryFilters *filtering.OrderQueryFilters) bool {
	ok := true
	count := 0

	if queryFilters.IdFilter != nil {
		ok = queryFilters.IdFilter.ApplyFilters(order.Id)
		if ok {
			count++
		}
	}
	if queryFilters.MarketFilter != nil {
		ok = queryFilters.MarketFilter.ApplyFilters(order.Market)
		if ok {
			count++
		}
	}
	if queryFilters.PartyFilter != nil {
		ok = queryFilters.PartyFilter.ApplyFilters(order.Party)
		if ok {
			count++
		}
	}
	if queryFilters.SideFilter != nil {
		ok = queryFilters.SideFilter.ApplyFilters(order.Side)
		if ok {
			count++
		}
	}
	if queryFilters.PriceFilter != nil {
		ok = queryFilters.PriceFilter.ApplyFilters(order.Price)
		if ok {
			count++
		}
	}
	if queryFilters.SizeFilter != nil {
		ok = queryFilters.SizeFilter.ApplyFilters(order.Size)
		if ok {
			count++
		}
	}
	if queryFilters.RemainingFilter != nil {
		ok = queryFilters.RemainingFilter.ApplyFilters(order.Remaining)
		if ok {
			count++
		}
	}
	if queryFilters.TypeFilter != nil {
		ok = queryFilters.TypeFilter.ApplyFilters(order.Type)
		if ok {
			count++
		}
	}
	if queryFilters.TimestampFilter != nil {
		ok = queryFilters.TimestampFilter.ApplyFilters(order.Timestamp)
		if ok {
			count++
		}
	}
	if queryFilters.StatusFilter != nil {
		ok = queryFilters.StatusFilter.ApplyFilters(order.Status)
		if ok {
			count++
		}
	}
	if queryFilters.ReferenceFilter != nil {
		ok = queryFilters.ReferenceFilter.ApplyFilters(order.Reference)
		if ok {
			count++
		}
	}

	if queryFilters.Operator == filtering.QueryFilterOperatorAnd {
		// If we AND all the queryFilters the counts should match
		// and if they do we have the exact match
		return count == queryFilters.Count()
	} else {
		// We are in an OR operation so if any of the queryFilters
		// have matched we can return true, false otherwise
		return ok
	}
}
