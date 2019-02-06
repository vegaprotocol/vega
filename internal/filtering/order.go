package filtering

type OrderQueryFilters struct {
	QueryFilterPaginated
	Operator QueryFilterOperator
	Open     bool

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
	ReferenceFilter *QueryFilter
}

func (o *OrderQueryFilters) Count() int {
	total := 0
	if o.IdFilter != nil {
		total++
	}
	if o.MarketFilter != nil {
		total++
	}
	if o.PartyFilter != nil {
		total++
	}
	if o.SideFilter != nil {
		total++
	}
	if o.PriceFilter != nil {
		total++
	}
	if o.SizeFilter != nil {
		total++
	}
	if o.RemainingFilter != nil {
		total++
	}
	if o.TypeFilter != nil {
		total++
	}
	if o.TimestampFilter != nil {
		total++
	}
	if o.StatusFilter != nil {
		total++
	}
	if o.ReferenceFilter != nil {
		total++
	}
	return total
}
