package gql

import (
	"github.com/pkg/errors"
	"vega/filters"
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
	if queryFilters.IDNeq != nil {
		id := *queryFilters.IDNeq
		holder.IdFilter = &filters.QueryFilter{
			Neq: id,
		}
		foundFilter = true
	}
	if queryFilters.Market != nil {
		// Todo(cdm): implement market-store/market-services validation lookup in nice-net
		err := validateMarket(queryFilters.Market)
		if err != nil {
			return false, err
		}
		holder.MarketFilter = &filters.QueryFilter{
			Eq: *queryFilters.Market,
		}
		foundFilter = true
	}
	if queryFilters.MarketNeq != nil {
		// Todo(cdm): implement market-store/market-services validation lookup in nice-net
		err := validateMarket(queryFilters.MarketNeq)
		if err != nil {
			return false, err
		}
		holder.MarketFilter = &filters.QueryFilter{
			Neq: *queryFilters.MarketNeq,
		}
		foundFilter = true
	}
	if queryFilters.Party != nil {
		// Todo(cdm): implement party-store/party-service validation in nice-net
		holder.PartyFilter = &filters.QueryFilter{
			Eq: *queryFilters.Party,
		}
		foundFilter = true
	}
	if queryFilters.PartyNeq != nil {
		// Todo(cdm): implement party-store/party-service validation in nice-net
		holder.PartyFilter = &filters.QueryFilter{
			Neq: *queryFilters.PartyNeq,
		}
		foundFilter = true
	}
	if queryFilters.Side != nil {
		side, err := parseSide(queryFilters.Side)
		if err != nil {
			return false, err
		}
		holder.SideFilter = &filters.QueryFilter{
			Eq: side,
		}
		foundFilter = true
	}
	if queryFilters.SideNeq != nil {
		side, err := parseSide(queryFilters.SideNeq)
		if err != nil {
			return false, err
		}
		holder.SideFilter = &filters.QueryFilter{
			Neq: side,
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
	if queryFilters.PriceNeq != nil {
		price, err := safeStringUint64(*queryFilters.PriceNeq)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filters.QueryFilter{
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
	if queryFilters.SizeNeq != nil {
		size, err := safeStringUint64(*queryFilters.SizeNeq)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &filters.QueryFilter{
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
	if queryFilters.RemainingNeq != nil {
		remaining, err := safeStringUint64(*queryFilters.RemainingNeq)
		if err != nil {
			return false, err
		}
		holder.RemainingFilter = &filters.QueryFilter{
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
	if queryFilters.TypeNeq != nil {
		orderType, err := parseOrderType(queryFilters.TypeNeq)
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
	if queryFilters.TimestampNeq != nil {
		timestamp, err := safeStringUint64(*queryFilters.TimestampNeq)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filters.QueryFilter{
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
	if queryFilters.StatusNeq != nil {
		orderStatus, err := parseOrderStatus(queryFilters.StatusNeq)
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
	if queryFilters.IDNeq != nil {
		id := *queryFilters.IDNeq
		holder.IdFilter = &filters.QueryFilter{
			Neq: id,
		}
		foundFilter = true
	}
	if queryFilters.Market != nil {

		// Todo(cdm): implement market-store/market-services validation lookup in nice-net
		err := validateMarket(queryFilters.Market)
		if err != nil {
			return false, err
		}

		holder.MarketFilter = &filters.QueryFilter{
			Eq: *queryFilters.Market,
		}
		foundFilter = true
	}
	if queryFilters.MarketNeq != nil {

		// Todo(cdm): implement market-store/market-services validation lookup in nice-net
		err := validateMarket(queryFilters.MarketNeq)
		if err != nil {
			return false, err
		}

		holder.MarketFilter = &filters.QueryFilter{
			Neq: *queryFilters.MarketNeq,
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
	if queryFilters.PriceNeq != nil {
		price, err := safeStringUint64(*queryFilters.PriceNeq)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &filters.QueryFilter{
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
	if queryFilters.SizeNeq != nil {
		size, err := safeStringUint64(*queryFilters.SizeNeq)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &filters.QueryFilter{
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
			Eq: *queryFilters.Buyer,
		}
		foundFilter = true
	}
	if queryFilters.BuyerNeq != nil {
		holder.BuyerFilter = &filters.QueryFilter{
			Neq: *queryFilters.BuyerNeq,
		}
		foundFilter = true
	}
	if queryFilters.Seller != nil {
		holder.SellerFilter = &filters.QueryFilter{
			Eq: *queryFilters.Seller,
		}
		foundFilter = true
	}
	if queryFilters.SellerNeq != nil {
		holder.SellerFilter = &filters.QueryFilter{
			Neq: *queryFilters.SellerNeq,
		}
		foundFilter = true
	}
	if queryFilters.Aggressor != nil {
		side, err := parseSide(queryFilters.Aggressor)
		if err != nil {
			return false, err
		}
		holder.AggressorFilter = &filters.QueryFilter{
			Eq: side,
		}
		foundFilter = true
	}
	if queryFilters.AggressorNeq != nil {
		side, err := parseSide(queryFilters.AggressorNeq)
		if err != nil {
			return false, err
		}
		holder.AggressorFilter = &filters.QueryFilter{
			Neq: side,
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
	if queryFilters.TimestampNeq != nil {
		timestamp, err := safeStringUint64(*queryFilters.TimestampNeq)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &filters.QueryFilter{
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
