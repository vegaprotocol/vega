package gql

import (
	"vega/common"
	"github.com/pkg/errors"
	"fmt"
)

func ParseOrderFilter(filters *OrderFilter, holder *common.OrderQueryFilters) (bool, error) {
	if filters == nil {
		return false, errors.New("filters must be set when calling ParseOrderFilter")
	}
	// In case the caller forgets to pass in a struct, we check and create the holder
	if holder == nil {
		holder = &common.OrderQueryFilters{}
	}
	// Match filters in GQL against the query filters in the api-services & data stores
	foundFilter := false
	if filters.ID != nil {
		id := *filters.ID

		fmt.Println("ID in filters gql: ", id)
		
		holder.IdFilter = &common.QueryFilter{
			Eq: id,
		}
		foundFilter = true
	}
	if filters.Id_neq != nil {
		id := *filters.Id_neq
		holder.IdFilter = &common.QueryFilter{
			Neq: id,
		}
		foundFilter = true
	}
	if filters.Market != nil {
		holder.MarketFilter = &common.QueryFilter{
			Eq: filters.Market,
		}
		foundFilter = true
	}
	if filters.Market_neq != nil {
		holder.MarketFilter = &common.QueryFilter{
			Neq: filters.Market_neq,
		}
		foundFilter = true
	}
	if filters.Party != nil {
		holder.PartyFilter = &common.QueryFilter{
			Eq: filters.Party,
		}
		foundFilter = true
	}
	if filters.Party_neq != nil {
		holder.PartyFilter = &common.QueryFilter{
			Neq: filters.Party_neq,
		}
		foundFilter = true
	}
	if filters.Side != nil {
		holder.SideFilter = &common.QueryFilter{
			Eq: filters.Side,
		}
		foundFilter = true
	}
	if filters.Side_neq != nil {
		holder.SideFilter = &common.QueryFilter{
			Neq: filters.Side_neq,
		}
		foundFilter = true
	}
	if filters.Price != nil {
		price, err := SafeStringUint64(*filters.Price)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &common.QueryFilter{
			Eq: price,
		}
		foundFilter = true
	}
	if filters.Price_neq != nil {
		price, err := SafeStringUint64(*filters.Price_neq)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &common.QueryFilter{
			Neq: price,
		}
		foundFilter = true
	}
	if filters.Price_from != nil && filters.Price_to != nil {
		lower, err := SafeStringUint64(*filters.Price_from)
		if err != nil {
			return false, err
		}
		upper, err := SafeStringUint64(*filters.Price_to)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &common.QueryFilter{
			FilterRange: &common.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if filters.Size != nil {
		size, err := SafeStringUint64(*filters.Size)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &common.QueryFilter{
			Eq: size,
		}
		foundFilter = true
	}
	if filters.Size_neq != nil {
		size, err := SafeStringUint64(*filters.Size_neq)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &common.QueryFilter{
			Neq: size,
		}
		foundFilter = true
	}
	if filters.Size_from != nil && filters.Size_to != nil {
		lower, err := SafeStringUint64(*filters.Size_from)
		if err != nil {
			return false, err
		}
		upper, err := SafeStringUint64(*filters.Size_to)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &common.QueryFilter{
			FilterRange: &common.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if filters.Remaining != nil {
		remaining, err := SafeStringUint64(*filters.Remaining)


		fmt.Println("Remaining in filters gql: ", remaining)

		if err != nil {
			return false, err
		}
		holder.RemainingFilter = &common.QueryFilter{
			Eq: remaining,
		}
		foundFilter = true
	}
	if filters.Remaining_neq != nil {
		remaining, err := SafeStringUint64(*filters.Remaining_neq)
		if err != nil {
			return false, err
		}
		holder.RemainingFilter = &common.QueryFilter{
			Neq: remaining,
		}
		foundFilter = true
	}
	if filters.Remaining_from != nil && filters.Remaining_to != nil {
		lower, err := SafeStringUint64(*filters.Remaining_from)
		if err != nil {
			return false, err
		}
		upper, err := SafeStringUint64(*filters.Remaining_to)
		if err != nil {
			return false, err
		}
		holder.RemainingFilter = &common.QueryFilter{
			FilterRange: &common.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if filters.Type != nil {
		orderType, err := ParseOrderType(filters.Type)
		if err != nil {
			return false, err
		}
		holder.TypeFilter = &common.QueryFilter{
			Eq: orderType,
		}
		foundFilter = true
	}
	if filters.Type_neq != nil {
		orderType, err := ParseOrderType(filters.Type_neq)
		if err != nil {
			return false, err
		}
		holder.TypeFilter = &common.QueryFilter{
			Neq: orderType,
		}
		foundFilter = true
	}
	if filters.Timestamp != nil {
		timestamp, err := SafeStringUint64(*filters.Timestamp)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &common.QueryFilter{
			Eq: timestamp,
		}
		foundFilter = true
	}
	if filters.Timestamp_neq != nil {
		timestamp, err := SafeStringUint64(*filters.Timestamp_neq)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &common.QueryFilter{
			Neq: timestamp,
		}
		foundFilter = true
	}
	if filters.Timestamp_from != nil && filters.Timestamp_to != nil {
		lower, err := SafeStringUint64(*filters.Timestamp_from)
		if err != nil {
			return false, err
		}
		upper, err := SafeStringUint64(*filters.Timestamp_to)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &common.QueryFilter{
			FilterRange: &common.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if filters.Status != nil {
		orderStatus, err := ParseOrderStatus(filters.Status)
		if err != nil {
			return false, err 
		}
		holder.StatusFilter = &common.QueryFilter{
			Eq: orderStatus,
		}
		foundFilter = true
	}
	if filters.Status_neq != nil {
		orderStatus, err := ParseOrderStatus(filters.Status_neq)
		if err != nil {
			return false, err
		}
		holder.StatusFilter = &common.QueryFilter{
			Neq: orderStatus,
		}
		foundFilter = true
	}
	return foundFilter, nil
}

func ParseTradeFilter(filters *TradeFilter, holder *common.TradeQueryFilters) (bool, error) {
	if filters == nil {
		return false, errors.New("filters must be set when calling ParseTradeFilter")
	}
	// In case the caller forgets to pass in a struct, we check and create the holder
	if holder == nil {
		holder = &common.TradeQueryFilters{}
	}
	// Match filters in GQL against the query filters in the api-services & data stores
	foundFilter := false
	if filters.ID != nil {
		id := *filters.ID
		holder.IdFilter = &common.QueryFilter{
			Eq: id,
		}
		foundFilter = true
	}
	if filters.Id_neq != nil {
		id := *filters.Id_neq
		holder.IdFilter = &common.QueryFilter{
			Neq: id,
		}
		foundFilter = true
	}
	if filters.Market != nil {
		holder.MarketFilter = &common.QueryFilter{
			Eq: filters.Market,
		}
		foundFilter = true
	}
	if filters.Market_neq != nil {
		holder.MarketFilter = &common.QueryFilter{
			Neq: filters.Market_neq,
		}
		foundFilter = true
	}
	if filters.Price != nil {
		price, err := SafeStringUint64(*filters.Price)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &common.QueryFilter{
			Eq: price,
		}
		foundFilter = true
	}
	if filters.Price_neq != nil {
		price, err := SafeStringUint64(*filters.Price_neq)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &common.QueryFilter{
			Neq: price,
		}
		foundFilter = true
	}
	if filters.Price_from != nil && filters.Price_to != nil {
		lower, err := SafeStringUint64(*filters.Price_from)
		if err != nil {
			return false, err
		}
		upper, err := SafeStringUint64(*filters.Price_to)
		if err != nil {
			return false, err
		}
		holder.PriceFilter = &common.QueryFilter{
			FilterRange: &common.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if filters.Size != nil {
		size, err := SafeStringUint64(*filters.Size)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &common.QueryFilter{
			Eq: size,
		}
		foundFilter = true
	}
	if filters.Size_neq != nil {
		size, err := SafeStringUint64(*filters.Size_neq)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &common.QueryFilter{
			Neq: size,
		}
		foundFilter = true
	}
	if filters.Size_from != nil && filters.Size_to != nil {
		lower, err := SafeStringUint64(*filters.Size_from)
		if err != nil {
			return false, err
		}
		upper, err := SafeStringUint64(*filters.Size_to)
		if err != nil {
			return false, err
		}
		holder.SizeFilter = &common.QueryFilter{
			FilterRange: &common.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	if filters.Buyer != nil {
		holder.BuyerFilter = &common.QueryFilter{
			Eq: filters.Buyer,
		}
		foundFilter = true
	}
	if filters.Buyer_neq != nil {
		holder.BuyerFilter = &common.QueryFilter{
			Neq: filters.Buyer_neq,
		}
		foundFilter = true
	}
	if filters.Seller != nil {
		holder.SellerFilter = &common.QueryFilter{
			Eq: filters.Seller,
		}
		foundFilter = true
	}
	if filters.Seller_neq != nil {
		holder.SellerFilter = &common.QueryFilter{
			Neq: filters.Seller_neq,
		}
		foundFilter = true
	}
	if filters.Aggressor != nil {
		holder.AggressorFilter = &common.QueryFilter{
			Eq: filters.Aggressor,
		}
		foundFilter = true
	}
	if filters.Aggressor_neq != nil {
		holder.AggressorFilter = &common.QueryFilter{
			Neq: filters.Aggressor_neq,
		}
		foundFilter = true
	}
	if filters.Timestamp != nil {
		timestamp, err := SafeStringUint64(*filters.Timestamp)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &common.QueryFilter{
			Eq: timestamp,
		}
		foundFilter = true
	}
	if filters.Timestamp_neq != nil {
		timestamp, err := SafeStringUint64(*filters.Timestamp_neq)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &common.QueryFilter{
			Neq: timestamp,
		}
		foundFilter = true
	}
	if filters.Timestamp_from != nil && filters.Timestamp_to != nil {
		lower, err := SafeStringUint64(*filters.Timestamp_from)
		if err != nil {
			return false, err
		}
		upper, err := SafeStringUint64(*filters.Timestamp_to)
		if err != nil {
			return false, err
		}
		holder.TimestampFilter = &common.QueryFilter{
			FilterRange: &common.QueryFilterRange{
				Lower: lower,
				Upper: upper,
			},
			Kind: "uint64",
		}
		foundFilter = true
	}
	return foundFilter, nil
}