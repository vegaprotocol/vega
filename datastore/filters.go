package datastore

import("vega/common")

// GetParamsLimitDefault should be used if no limit is specified
// when working with the GetParams struct.
const GetParamsLimitDefault = uint64(1844674407370955161)

type GetOrderParams struct {
	common.QueryFilterPaginated

	Limit           uint64

	MarketFilter    *common.QueryFilter
	PartyFilter     *common.QueryFilter
	SideFilter      *common.QueryFilter
	PriceFilter     *common.QueryFilter
	SizeFilter      *common.QueryFilter
	RemainingFilter *common.QueryFilter
	TypeFilter      *common.QueryFilter
	TimestampFilter *common.QueryFilter
	RiskFactor      *common.QueryFilter
	StatusFilter    *common.QueryFilter
}

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

func applyOrderFilter(order Order, params GetOrderParams) bool {
	var ok = true

	if params.MarketFilter != nil {
		ok = params.MarketFilter.ApplyFilters(order.Market)
	}

	if params.PartyFilter != nil {
		ok = params.PartyFilter.ApplyFilters(order.Party)
	}

	if params.SideFilter != nil {
		ok = params.SideFilter.ApplyFilters(order.Side)
	}

	if params.PriceFilter != nil {
		ok = params.PriceFilter.ApplyFilters(order.Price)
	}

	if params.SizeFilter != nil {
		ok = params.SizeFilter.ApplyFilters(order.Size)
	}

	if params.RemainingFilter != nil {
		ok = params.RemainingFilter.ApplyFilters(order.Remaining)
	}

	if params.TypeFilter != nil {
		ok = params.TypeFilter.ApplyFilters(order.Type)
	}

	if params.TimestampFilter != nil {
		ok = params.TimestampFilter.ApplyFilters(order.Timestamp)
	}

	if params.RiskFactor != nil {
		ok = params.RiskFactor.ApplyFilters(order.RiskFactor)
	}

	if params.StatusFilter != nil {
		ok = params.StatusFilter.ApplyFilters(order.Status)
	}

	return ok
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

