package gql

import (
	"code.vegaprotocol.io/vega/internal/filtering"

	"github.com/pkg/errors"
)

func ParseOrderFilter(queryFilters *OrderFilter, holder *filtering.OrderQueryFilters) (bool, error) {
	if queryFilters == nil {
		return false, errors.New("filters must be set when calling ParseOrderFilter")
	}
	// In case the caller forgets to pass in a struct, we check and create the holder
	if holder == nil {
		holder = &filtering.OrderQueryFilters{}
	}
	// Match filters in GQL against the query filters in the api-services & data stores
	foundFilter := false
	if queryFilters.ID != nil {
		id := *queryFilters.ID
		holder.IdFilter = &filtering.QueryFilter{
			Eq: id,
		}
		foundFilter = true
	}
	if queryFilters.IDNeq != nil {
		id := *queryFilters.IDNeq
		holder.IdFilter = &filtering.QueryFilter{
			Neq: id,
		}
		foundFilter = true
	}
	if queryFilters.Market != nil {
		holder.MarketFilter = &filtering.QueryFilter{
			Eq: *queryFilters.Market,
		}
		foundFilter = true
	}
	if queryFilters.MarketNeq != nil {
		holder.MarketFilter = &filtering.QueryFilter{
			Neq: *queryFilters.MarketNeq,
		}
		foundFilter = true
	}
	if queryFilters.Party != nil {
		holder.PartyFilter = &filtering.QueryFilter{
			Eq: *queryFilters.Party,
		}
		foundFilter = true
	}
	if queryFilters.PartyNeq != nil {
		holder.PartyFilter = &filtering.QueryFilter{
			Neq: *queryFilters.PartyNeq,
		}
		foundFilter = true
	}
	if queryFilters.Side != nil {
		side, err := parseSide(queryFilters.Side)
		if err != nil {
			return false, err
		}
		holder.SideFilter = &filtering.QueryFilter{
			Eq: side,
		}
		foundFilter = true
	}
	if queryFilters.SideNeq != nil {
		side, err := parseSide(queryFilters.SideNeq)
		if err != nil {
			return false, err
		}
		holder.SideFilter = &filtering.QueryFilter{
			Neq: side,
		}
		foundFilter = true
	}
	if queryFilters.Price != nil {
		price, err := safeStringUint64(*queryFilters.Price)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filtering.QueryFilter{
			Eq: price,
		}
		foundFilter = true
	}
	if queryFilters.PriceNeq != nil {
		price, err := safeStringUint64(*queryFilters.PriceNeq)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filtering.QueryFilter{
			Neq: price,
		}
		foundFilter = true
	}
	if queryFilters.PriceFrom != nil && queryFilters.PriceTo != nil {
		lower, err := safeStringUint64(*queryFilters.PriceFrom)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.PriceTo)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filtering.QueryFilter{
			FilterRange: &filtering.QueryFilterRange{
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
		holder.SizeFilter = &filtering.QueryFilter{
			Eq: size,
		}
		foundFilter = true
	}
	if queryFilters.SizeNeq != nil {
		size, err := safeStringUint64(*queryFilters.SizeNeq)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &filtering.QueryFilter{
			Neq: size,
		}
		foundFilter = true
	}
	if queryFilters.SizeFrom != nil && queryFilters.SizeTo != nil {
		lower, err := safeStringUint64(*queryFilters.SizeFrom)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.SizeTo)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &filtering.QueryFilter{
			FilterRange: &filtering.QueryFilterRange{
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
		holder.RemainingFilter = &filtering.QueryFilter{
			Eq: remaining,
		}
		foundFilter = true
	}
	if queryFilters.RemainingNeq != nil {
		remaining, err := safeStringUint64(*queryFilters.RemainingNeq)
		if err != nil {
			return false, err
		}
		holder.RemainingFilter = &filtering.QueryFilter{
			Neq: remaining,
		}
		foundFilter = true
	}
	if queryFilters.RemainingFrom != nil && queryFilters.RemainingTo != nil {
		lower, err := safeStringUint64(*queryFilters.RemainingFrom)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.RemainingTo)
		if err != nil {
			return false, err
		}
		holder.RemainingFilter = &filtering.QueryFilter{
			FilterRange: &filtering.QueryFilterRange{
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
		holder.TypeFilter = &filtering.QueryFilter{
			Eq: orderType,
		}
		foundFilter = true
	}
	if queryFilters.TypeNeq != nil {
		orderType, err := parseOrderType(queryFilters.TypeNeq)
		if err != nil {
			return false, err
		}
		holder.TypeFilter = &filtering.QueryFilter{
			Neq: orderType,
		}
		foundFilter = true
	}
	if queryFilters.Timestamp != nil {
		timestamp, err := safeStringUint64(*queryFilters.Timestamp)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filtering.QueryFilter{
			Eq: timestamp,
		}
		foundFilter = true
	}
	if queryFilters.TimestampNeq != nil {
		timestamp, err := safeStringUint64(*queryFilters.TimestampNeq)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filtering.QueryFilter{
			Neq: timestamp,
		}
		foundFilter = true
	}
	if queryFilters.TimestampFrom != nil && queryFilters.TimestampTo != nil {
		lower, err := safeStringUint64(*queryFilters.TimestampFrom)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.TimestampTo)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filtering.QueryFilter{
			FilterRange: &filtering.QueryFilterRange{
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
		holder.StatusFilter = &filtering.QueryFilter{
			Eq: orderStatus,
		}
		foundFilter = true
	}
	if queryFilters.StatusNeq != nil {
		orderStatus, err := parseOrderStatus(queryFilters.StatusNeq)
		if err != nil {
			return false, err
		}
		holder.StatusFilter = &filtering.QueryFilter{
			Neq: orderStatus,
		}
		foundFilter = true
	}
	if queryFilters.Open != nil {
		holder.Open = *queryFilters.Open
	}
	return foundFilter, nil
}

func ParseTradeFilter(queryFilters *TradeFilter, holder *filtering.TradeQueryFilters) (bool, error) {
	if queryFilters == nil {
		return false, errors.New("filters must be set when calling ParseTradeFilter")
	}
	// In case the caller forgets to pass in a struct, we check and create the holder
	if holder == nil {
		holder = &filtering.TradeQueryFilters{}
	}
	// Match filters in GQL against the query filters in the api-services & data stores
	foundFilter := false
	if queryFilters.ID != nil {
		id := *queryFilters.ID
		holder.IdFilter = &filtering.QueryFilter{
			Eq: id,
		}
		foundFilter = true
	}
	if queryFilters.IDNeq != nil {
		id := *queryFilters.IDNeq
		holder.IdFilter = &filtering.QueryFilter{
			Neq: id,
		}
		foundFilter = true
	}
	if queryFilters.Market != nil {
		holder.MarketFilter = &filtering.QueryFilter{
			Eq: *queryFilters.Market,
		}
		foundFilter = true
	}
	if queryFilters.MarketNeq != nil {
		holder.MarketFilter = &filtering.QueryFilter{
			Neq: *queryFilters.MarketNeq,
		}
		foundFilter = true
	}
	if queryFilters.Price != nil {
		price, err := safeStringUint64(*queryFilters.Price)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filtering.QueryFilter{
			Eq: price,
		}
		foundFilter = true
	}
	if queryFilters.PriceNeq != nil {
		price, err := safeStringUint64(*queryFilters.PriceNeq)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filtering.QueryFilter{
			Neq: price,
		}
		foundFilter = true
	}
	if queryFilters.PriceFrom != nil && queryFilters.PriceTo != nil {
		lower, err := safeStringUint64(*queryFilters.PriceFrom)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.PriceTo)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filtering.QueryFilter{
			FilterRange: &filtering.QueryFilterRange{
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
		holder.SizeFilter = &filtering.QueryFilter{
			Eq: size,
		}
		foundFilter = true
	}
	if queryFilters.SizeNeq != nil {
		size, err := safeStringUint64(*queryFilters.SizeNeq)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &filtering.QueryFilter{
			Neq: size,
		}
		foundFilter = true
	}
	if queryFilters.SizeFrom != nil && queryFilters.SizeTo != nil {
		lower, err := safeStringUint64(*queryFilters.SizeFrom)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.SizeTo)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &filtering.QueryFilter{
			FilterRange: &filtering.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if queryFilters.Buyer != nil {
		holder.BuyerFilter = &filtering.QueryFilter{
			Eq: *queryFilters.Buyer,
		}
		foundFilter = true
	}
	if queryFilters.BuyerNeq != nil {
		holder.BuyerFilter = &filtering.QueryFilter{
			Neq: *queryFilters.BuyerNeq,
		}
		foundFilter = true
	}
	if queryFilters.Seller != nil {
		holder.SellerFilter = &filtering.QueryFilter{
			Eq: *queryFilters.Seller,
		}
		foundFilter = true
	}
	if queryFilters.SellerNeq != nil {
		holder.SellerFilter = &filtering.QueryFilter{
			Neq: *queryFilters.SellerNeq,
		}
		foundFilter = true
	}
	if queryFilters.Aggressor != nil {
		side, err := parseSide(queryFilters.Aggressor)
		if err != nil {
			return false, err
		}
		holder.AggressorFilter = &filtering.QueryFilter{
			Eq: side,
		}
		foundFilter = true
	}
	if queryFilters.AggressorNeq != nil {
		side, err := parseSide(queryFilters.AggressorNeq)
		if err != nil {
			return false, err
		}
		holder.AggressorFilter = &filtering.QueryFilter{
			Neq: side,
		}
		foundFilter = true
	}
	if queryFilters.Timestamp != nil {
		timestamp, err := safeStringUint64(*queryFilters.Timestamp)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filtering.QueryFilter{
			Eq: timestamp,
		}
		foundFilter = true
	}
	if queryFilters.TimestampNeq != nil {
		timestamp, err := safeStringUint64(*queryFilters.TimestampNeq)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filtering.QueryFilter{
			Neq: timestamp,
		}
		foundFilter = true
	}
	if queryFilters.TimestampFrom != nil && queryFilters.TimestampTo != nil {
		lower, err := safeStringUint64(*queryFilters.TimestampFrom)
		if err != nil {
			return false, err
		}
		upper, err := safeStringUint64(*queryFilters.TimestampTo)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filtering.QueryFilter{
			FilterRange: &filtering.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	return foundFilter, nil
}
