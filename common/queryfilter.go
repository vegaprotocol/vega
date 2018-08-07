package common

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

type TradeQueryFilters struct {
	QueryFilterPaginated

	MarketFilter    *QueryFilter
	PriceFilter     *QueryFilter
	SizeFilter      *QueryFilter
	BuyerFilter     *QueryFilter
	SellerFilter    *QueryFilter
	AggressorFilter *QueryFilter
	TimestampFilter *QueryFilter
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


