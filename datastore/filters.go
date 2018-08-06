package datastore

import("vega/common")

// GetParamsLimitDefault should be used if no limit is specified
// when working with the GetParams struct.
const GetParamsLimitDefault = uint64(1844674407370955161)

// GetParams is used for optional parameters that can be passed
// into the datastores when querying for records.
type GetOrderParams struct {
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
		ok = apply(order.Market, params.MarketFilter)
	}

	if params.PartyFilter != nil {
		ok = apply(order.Party, params.PartyFilter)
	}

	if params.SideFilter != nil {
		ok = apply(order.Side, params.SideFilter)
	}

	if params.PriceFilter != nil {
		ok = apply(order.Price, params.PriceFilter)
	}

	if params.SizeFilter != nil {
		ok = apply(order.Size, params.SizeFilter)
	}

	if params.RemainingFilter != nil {
		ok = apply(order.Remaining, params.RemainingFilter)
	}

	if params.TypeFilter != nil {
		ok = apply(order.Type, params.TypeFilter)
	}

	if params.TimestampFilter != nil {
		ok = apply(order.Timestamp, params.TimestampFilter)
	}

	if params.RiskFactor != nil {
		ok = apply(order.RiskFactor, params.RiskFactor)
	}

	if params.StatusFilter != nil {
		ok = apply(order.Status, params.StatusFilter)
	}

	return ok
}

func applyTradeFilter(trade Trade, params GetTradeParams) bool {
	var ok = true

	if params.MarketFilter != nil {
		ok = apply(trade.Market, params.MarketFilter)
	}

	if params.PriceFilter != nil {
		ok = apply(trade.Price, params.PriceFilter)
	}

	if params.SizeFilter != nil {
		ok = apply(trade.Size, params.SizeFilter)
	}

	if params.BuyerFilter != nil {
		ok = apply(trade.Buyer, params.BuyerFilter)
	}

	if params.SellerFilter != nil {
		ok = apply(trade.Seller, params.SellerFilter)
	}

	if params.AggressorFilter != nil {
		ok = apply(trade.Aggressor, params.AggressorFilter)
	}

	if params.TimestampFilter != nil {
		ok = apply(trade.Timestamp, params.TimestampFilter)
	}

	return ok
}

func apply(value interface{}, params *common.QueryFilter) bool {
	if params.FilterRange != nil {
		return params.ApplyRangeFilter(value, params.FilterRange, params.Kind)
	}

	if params.Eq != nil {
		return params.ApplyEqualFilter(value, params.Eq)
	}

	if params.Neq != nil {
		return params.ApplyNotEqualFilter(value, params.Neq)
	}
	return false
}
