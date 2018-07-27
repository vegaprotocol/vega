package datastore

import "github.com/golang/go/src/pkg/fmt"

// GetParamsLimitDefault should be used if no limit is specified
// when working with the GetParams struct.
const GetParamsLimitDefault = uint64(1844674407370955161)

// GetParams is used for optional parameters that can be passed
// into the datastores when querying for records.
type GetOrderParams struct {
	Limit           uint64
	MarketFilter    *QueryFilter
	PartyFilter     *QueryFilter
	SideFilter      *QueryFilter
	PriceFilter     *QueryFilter
	SizeFilter      *QueryFilter
	RemainingFilter *QueryFilter
	TypeFilter      *QueryFilter
	TimestampFilter *QueryFilter
	RiskFactor      *QueryFilter
	StatusFilter    *QueryFilter
}

type GetTradeParams struct {
	Limit  uint64
	Size   uint64
	Buyer  string
	Seller string
}

type QueryFilterType int

type QueryFilter struct {
	filterRange *Range
	neq         interface{}
	eq          interface{}
	kind        string
}

type Range struct {
	Lower interface{}
	Upper interface{}
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

func apply(value interface{}, params *QueryFilter) bool {
	fmt.Printf("params: %+v\n", params)
	if params.filterRange != nil {
		return applyRangeFilter(value, params.filterRange, params.kind)
	}

	if params.eq != nil {
		return applyEqualFilter(value, params.eq)
	}

	if params.neq != nil {
		return applyNotEqualFilter(value, params.neq)
	}
	return false
}

func applyRangeFilter(value interface{}, r *Range, kind string) bool {
	fmt.Printf("params: %+v\n", r)
	fmt.Printf("value: %+v\n", value.(uint64))
	if kind == "uint64" {
		if r.Lower.(uint64) <= value.(uint64) && value.(uint64) <= r.Upper.(uint64) {
			fmt.Printf("TRUE")
			return true
		}
	}

	// add new kind here
	//if kind == "NEW_KIND" {
	//		...
	//}

	return false
}

func applyEqualFilter(value interface{}, eq interface{}) bool {
	if eq == value {
		return true
	}
	return false
}

func applyNotEqualFilter(value interface{}, neq interface{}) bool {
	if neq != value {
		return true
	}
	return false
}
