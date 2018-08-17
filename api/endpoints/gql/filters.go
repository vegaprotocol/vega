package gql

import (
	"vega/filters"
	"github.com/pkg/errors"
)

func ParseOrderFilter(queryFilters *OrderFilter, holder *filters.OrderQueryFilters) (bool, error) {
	if queryFilters == nil {
		return false, errors.New("filters must be set when calling ParseOrderFilter")
	}
	// In case the caller forgets to pass in a struct, we check and create the holder
	if holder == nil {
		holder = &filters.OrderQueryFilters{}
	}
	// Match filters in GQL against the query filters in the api-services & data stores
	foundFilter := false
	if queryFilters.ID != nil {
		id := *queryFilters.ID
		holder.IdFilter = &filters.QueryFilter{
			Eq: id,
		}
		foundFilter = true
	}
	if queryFilters.Id_neq != nil {
		id := *queryFilters.Id_neq
		holder.IdFilter = &filters.QueryFilter{
			Neq: id,
		}
		foundFilter = true
	}
	if queryFilters.Market != nil {
		holder.MarketFilter = &filters.QueryFilter{
			Eq: *queryFilters.Market,
		}
		foundFilter = true
	}
	if queryFilters.Market_neq != nil {
		holder.MarketFilter = &filters.QueryFilter{
			Neq: *queryFilters.Market_neq,
		}
		foundFilter = true
	}
	if queryFilters.Party != nil {
		holder.PartyFilter = &filters.QueryFilter{
			Eq: *queryFilters.Party,
		}
		foundFilter = true
	}
	if queryFilters.Party_neq != nil {
		holder.PartyFilter = &filters.QueryFilter{
			Neq: *queryFilters.Party_neq,
		}
		foundFilter = true
	}
	if queryFilters.Side != nil {
		holder.SideFilter = &filters.QueryFilter{
			Eq: queryFilters.Side,
		}
		foundFilter = true
	}
	if queryFilters.Side_neq != nil {
		holder.SideFilter = &filters.QueryFilter{
			Neq: queryFilters.Side_neq,
		}
		foundFilter = true
	}
	if queryFilters.Price != nil {
		price, err := safeStringUint64(*queryFilters.Price)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filters.QueryFilter{
			Eq: price,
		}
		foundFilter = true
	}
	if queryFilters.Price_neq != nil {
		price, err := safeStringUint64(*queryFilters.Price_neq)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filters.QueryFilter{
			Neq: price,
		}
		foundFilter = true
	}
	if queryFilters.Price_from != nil && queryFilters.Price_to != nil {
		lower, err := safeStringUint64(*queryFilters.Price_from)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.Price_to)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filters.QueryFilter{
			FilterRange: &filters.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if queryFilters.Size != nil {
		size, err := safeStringUint64(*queryFilters.Size)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &filters.QueryFilter{
			Eq: size,
		}
		foundFilter = true
	}
	if queryFilters.Size_neq != nil {
		size, err := safeStringUint64(*queryFilters.Size_neq)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &filters.QueryFilter{
			Neq: size,
		}
		foundFilter = true
	}
	if queryFilters.Size_from != nil && queryFilters.Size_to != nil {
		lower, err := safeStringUint64(*queryFilters.Size_from)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.Size_to)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &filters.QueryFilter{
			FilterRange: &filters.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if queryFilters.Remaining != nil {
		remaining, err := safeStringUint64(*queryFilters.Remaining)
		if err != nil {
			return false, err
		}
		holder.RemainingFilter = &filters.QueryFilter{
			Eq: remaining,
		}
		foundFilter = true
	}
	if queryFilters.Remaining_neq != nil {
		remaining, err := safeStringUint64(*queryFilters.Remaining_neq)
		if err != nil {
			return false, err
		}
		holder.RemainingFilter = &filters.QueryFilter{
			Neq: remaining,
		}
		foundFilter = true
	}
	if queryFilters.Remaining_from != nil && queryFilters.Remaining_to != nil {
		lower, err := safeStringUint64(*queryFilters.Remaining_from)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.Remaining_to)
		if err != nil {
			return false, err
		}
		holder.RemainingFilter = &filters.QueryFilter{
			FilterRange: &filters.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if queryFilters.Type != nil {
		orderType, err := parseOrderType(queryFilters.Type)
		if err != nil {
			return false, err
		}
		holder.TypeFilter = &filters.QueryFilter{
			Eq: orderType,
		}
		foundFilter = true
	}
	if queryFilters.Type_neq != nil {
		orderType, err := parseOrderType(queryFilters.Type_neq)
		if err != nil {
			return false, err
		}
		holder.TypeFilter = &filters.QueryFilter{
			Neq: orderType,
		}
		foundFilter = true
	}
	if queryFilters.Timestamp != nil {
		timestamp, err := safeStringUint64(*queryFilters.Timestamp)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filters.QueryFilter{
			Eq: timestamp,
		}
		foundFilter = true
	}
	if queryFilters.Timestamp_neq != nil {
		timestamp, err := safeStringUint64(*queryFilters.Timestamp_neq)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filters.QueryFilter{
			Neq: timestamp,
		}
		foundFilter = true
	}
	if queryFilters.Timestamp_from != nil && queryFilters.Timestamp_to != nil {
		lower, err := safeStringUint64(*queryFilters.Timestamp_from)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.Timestamp_to)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filters.QueryFilter{
			FilterRange: &filters.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if queryFilters.Status != nil {
		orderStatus, err := parseOrderStatus(queryFilters.Status)
		if err != nil {
			return false, err 
		}
		holder.StatusFilter = &filters.QueryFilter{
			Eq: orderStatus,
		}
		foundFilter = true
	}
	if queryFilters.Status_neq != nil {
		orderStatus, err := parseOrderStatus(queryFilters.Status_neq)
		if err != nil {
			return false, err
		}
		holder.StatusFilter = &filters.QueryFilter{
			Neq: orderStatus,
		}
		foundFilter = true
	}
	if queryFilters.Open != nil {
		holder.Open = *queryFilters.Open
	}
	return foundFilter, nil
}

func ParseTradeFilter(queryFilters *TradeFilter, holder *filters.TradeQueryFilters) (bool, error) {
	if queryFilters == nil {
		return false, errors.New("filters must be set when calling ParseTradeFilter")
	}
	// In case the caller forgets to pass in a struct, we check and create the holder
	if holder == nil {
		holder = &filters.TradeQueryFilters{}
	}
	// Match filters in GQL against the query filters in the api-services & data stores
	foundFilter := false
	if queryFilters.ID != nil {
		id := *queryFilters.ID
		holder.IdFilter = &filters.QueryFilter{
			Eq: id,
		}
		foundFilter = true
	}
	if queryFilters.Id_neq != nil {
		id := *queryFilters.Id_neq
		holder.IdFilter = &filters.QueryFilter{
			Neq: id,
		}
		foundFilter = true
	}
	if queryFilters.Market != nil {
		holder.MarketFilter = &filters.QueryFilter{
			Eq: queryFilters.Market,
		}
		foundFilter = true
	}
	if queryFilters.Market_neq != nil {
		holder.MarketFilter = &filters.QueryFilter{
			Neq: queryFilters.Market_neq,
		}
		foundFilter = true
	}
	if queryFilters.Price != nil {
		price, err := safeStringUint64(*queryFilters.Price)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filters.QueryFilter{
			Eq: price,
		}
		foundFilter = true
	}
	if queryFilters.Price_neq != nil {
		price, err := safeStringUint64(*queryFilters.Price_neq)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filters.QueryFilter{
			Neq: price,
		}
		foundFilter = true
	}
	if queryFilters.Price_from != nil && queryFilters.Price_to != nil {
		lower, err := safeStringUint64(*queryFilters.Price_from)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.Price_to)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filters.QueryFilter{
			FilterRange: &filters.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if queryFilters.Size != nil {
		size, err := safeStringUint64(*queryFilters.Size)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &filters.QueryFilter{
			Eq: size,
		}
		foundFilter = true
	}
	if queryFilters.Size_neq != nil {
		size, err := safeStringUint64(*queryFilters.Size_neq)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &filters.QueryFilter{
			Neq: size,
		}
		foundFilter = true
	}
	if queryFilters.Size_from != nil && queryFilters.Size_to != nil {
		lower, err := safeStringUint64(*queryFilters.Size_from)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.Size_to)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &filters.QueryFilter{
			FilterRange: &filters.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if queryFilters.Buyer != nil {
		holder.BuyerFilter = &filters.QueryFilter{
			Eq: queryFilters.Buyer,
		}
		foundFilter = true
	}
	if queryFilters.Buyer_neq != nil {
		holder.BuyerFilter = &filters.QueryFilter{
			Neq: queryFilters.Buyer_neq,
		}
		foundFilter = true
	}
	if queryFilters.Seller != nil {
		holder.SellerFilter = &filters.QueryFilter{
			Eq: queryFilters.Seller,
		}
		foundFilter = true
	}
	if queryFilters.Seller_neq != nil {
		holder.SellerFilter = &filters.QueryFilter{
			Neq: queryFilters.Seller_neq,
		}
		foundFilter = true
	}
	if queryFilters.Aggressor != nil {
		holder.AggressorFilter = &filters.QueryFilter{
			Eq: queryFilters.Aggressor,
		}
		foundFilter = true
	}
	if queryFilters.Aggressor_neq != nil {
		holder.AggressorFilter = &filters.QueryFilter{
			Neq: queryFilters.Aggressor_neq,
		}
		foundFilter = true
	}
	if queryFilters.Timestamp != nil {
		timestamp, err := safeStringUint64(*queryFilters.Timestamp)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filters.QueryFilter{
			Eq: timestamp,
		}
		foundFilter = true
	}
	if queryFilters.Timestamp_neq != nil {
		timestamp, err := safeStringUint64(*queryFilters.Timestamp_neq)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filters.QueryFilter{
			Neq: timestamp,
		}
		foundFilter = true
	}
	if queryFilters.Timestamp_from != nil && queryFilters.Timestamp_to != nil {
		lower, err := safeStringUint64(*queryFilters.Timestamp_from)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.Timestamp_to)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filters.QueryFilter{
			FilterRange: &filters.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	return foundFilter, nil
}