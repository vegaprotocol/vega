package filtering

type TradeQueryFilters struct {
	QueryFilterPaginated
	Operator QueryFilterOperator

	IdFilter        *QueryFilter
	MarketFilter    *QueryFilter
	PriceFilter     *QueryFilter
	SizeFilter      *QueryFilter
	BuyerFilter     *QueryFilter
	SellerFilter    *QueryFilter
	AggressorFilter *QueryFilter
	TimestampFilter *QueryFilter
	BuyOrderFilter  *QueryFilter
	SellOrderFilter *QueryFilter
}

func (o *TradeQueryFilters) Count() int {
	total := 0
	if o.IdFilter != nil {
		total++
	}
	if o.MarketFilter != nil {
		total++
	}
	if o.PriceFilter != nil {
		total++
	}
	if o.SizeFilter != nil {
		total++
	}
	if o.BuyerFilter != nil {
		total++
	}
	if o.SellerFilter != nil {
		total++
	}
	if o.AggressorFilter != nil {
		total++
	}
	if o.TimestampFilter != nil {
		total++
	}
	if o.BuyOrderFilter != nil {
		total++
	}
	if o.SellOrderFilter != nil {
		total++
	}
	return total
}
