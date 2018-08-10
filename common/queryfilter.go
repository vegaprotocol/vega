package common

const QueryFilterOperatorAnd QueryFilterOperator = 0
const QueryFilterOperatorOr QueryFilterOperator = 1

type QueryFilterOperator int8

type QueryFilterType int

type QueryFilterRange struct {
	Lower interface{}
	Upper interface{}
}

type QueryFilterPaginated struct {
	First *uint64
	Last  *uint64
	Skip  *uint64
}

type OrderQueryFilters struct {
	QueryFilterPaginated
	Operator        QueryFilterOperator
	Open            bool

	IdFilter        *QueryFilter
	MarketFilter    *QueryFilter
	PartyFilter     *QueryFilter
	SideFilter      *QueryFilter
	PriceFilter     *QueryFilter
	SizeFilter      *QueryFilter
	RemainingFilter *QueryFilter
	TypeFilter      *QueryFilter
	TimestampFilter *QueryFilter
	StatusFilter    *QueryFilter
}

func (o *OrderQueryFilters) Count() int {
	len := 0
	if o.IdFilter != nil {
		len++
	}
	if o.MarketFilter != nil {
		len++
	}
	if o.PartyFilter != nil {
		len++
	}
	if o.SideFilter != nil {
		len++
	}
	if o.PriceFilter != nil {
		len++
	}
	if o.SizeFilter != nil {
		len++
	}
	if o.RemainingFilter != nil {
		len++
	}
	if o.TypeFilter != nil {
		len++
	}
	if o.TimestampFilter != nil {
		len++
	}
	if o.StatusFilter != nil {
		len++
	}
	return len
}

type TradeQueryFilters struct {
	QueryFilterPaginated
	Operator        QueryFilterOperator

	IdFilter        *QueryFilter
	MarketFilter    *QueryFilter
	PriceFilter     *QueryFilter
	SizeFilter      *QueryFilter
	BuyerFilter     *QueryFilter
	SellerFilter    *QueryFilter
	AggressorFilter *QueryFilter
	TimestampFilter *QueryFilter
}

func (o *TradeQueryFilters) Count() int {
	len := 0
	if o.IdFilter != nil {
		len++
	}
	if o.MarketFilter != nil {
		len++
	}
	if o.PriceFilter != nil {
		len++
	}
	if o.SizeFilter != nil {
		len++
	}
	if o.BuyerFilter != nil {
		len++
	}
	if o.SellerFilter != nil {
		len++
	}
	if o.AggressorFilter != nil {
		len++
	}
	if o.TimestampFilter != nil {
		len++
	}
	return len
}


type QueryFilter struct {
	FilterRange *QueryFilterRange
	Neq         interface{}
	Eq          interface{}
	Kind        string
}

func (q *QueryFilter) ApplyFilters(value interface{}) bool {
	if q.FilterRange != nil {
		return q.ApplyRangeFilter(value, q.FilterRange, q.Kind)
	}
	if q.Eq != nil {
		return q.ApplyEqualFilter(value, q.Eq)
	}
	if q.Neq != nil {
		return q.ApplyNotEqualFilter(value, q.Neq)
	}
	return false
}

func (q *QueryFilter) ApplyRangeFilter(value interface{}, r *QueryFilterRange, kind string) bool {
	if kind == "uint64" {
		if r.Lower.(uint64) <= value.(uint64) && value.(uint64) <= r.Upper.(uint64) {
			return true
		}
	}

	// add new kind here
	//if kind == "NEW_KIND" {
	//		...
	//}

	return false
}

func (q *QueryFilter) ApplyEqualFilter(value interface{}, eq interface{}) bool {
	if eq == value {
		return true
	}
	return false
}

func (q *QueryFilter) ApplyNotEqualFilter(value interface{}, neq interface{}) bool {
	if neq != value {
		return true
	}
	return false
}


