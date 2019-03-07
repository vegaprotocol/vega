package storage

import (
	types "code.vegaprotocol.io/vega/proto"
)

// MarketDepth provides a way to update and read the current state of Depth of Market.
type MarketDepth interface {
	// Update the market depth with the given order information. If the order already exists at a price level
	// it will be updated. Note: The total cumulative volume for the market depth is calculated elsewhere.
	Update(order types.Order)
	// The buy side price levels (and additional information such as orders, remaining volumes) for the market.
	BuySide() []MarketDepthLevel
	// The sell side price levels (and additional information such as orders, remaining volumes) for the market.
	SellSide() []MarketDepthLevel
}

// marketDepth stores the price levels on both buy side and sell side for a particular market.
type marketDepth struct {
	Name string
	Buy  []MarketDepthLevel
	Sell []MarketDepthLevel
}

// MarketDepthLevel keeps information on the price level and a map of the remaining for each order at that level.
type MarketDepthLevel struct {
	types.PriceLevel                   // price level details
	orders           map[string]uint64 // map of order.Id => remaining value
}

// NewMarketDepth creates a new market depth implementation for the given market name. With multiple markets,
// this initialiser can be used to create a market depth structure 1:1 per market.
func NewMarketDepth(name string) MarketDepth {
	return &marketDepth{Name: name}
}

// Update the market depth with the given order information. If the order already exists at a price level
// it will be updated. Note: The total cumulative volume for the market depth is calculated elsewhere.
func (md *marketDepth) Update(order types.Order) {
	if order.Side == types.Side_Buy {
		md.updateBuySide(order)
	} else {
		md.updateSellSide(order)
	}
}

// Called by Update to do the iteration over price levels and update the buy side of the market depth with
// order information. We now use a map of orderId => remaining to no longer need a DB lookup elsewhere.
func (md *marketDepth) updateBuySide(order types.Order) {
	var at = -1
	orderInvalid := md.isInvalid(order)

	// search through existing price/depth levels to find a position to insert
	found := false
	for idx, priceLevel := range md.Buy {
		if priceLevel.Price > order.Price {
			continue
		}
		if priceLevel.Price == order.Price {
			found = true
		}
		at = idx
		break
	}

	// check if the price/depth level was found for order price (to update existing total)
	if found {
		delta := uint64(0)

		// check if there's a previous order at this price level
		if existingRemaining, ok := md.Buy[at].orders[order.Id]; ok {
			// check if order is now fully filled or not trade-able status
			if orderInvalid {
				// order doesn't exist for price so remove from existing map
				md.Buy[at].Volume -= existingRemaining
				md.Buy[at].NumberOfOrders--
				delete(md.Buy[at].orders, order.Id)
			} else {
				delta = md.Buy[at].orders[order.Id] - order.Remaining
				md.Buy[at].orders[order.Id] = order.Remaining
				md.Buy[at].Volume -= delta
			}
		} else if !orderInvalid {
			md.Buy[at].orders[order.Id] = order.Remaining
			md.Buy[at].Volume += order.Remaining
			md.Buy[at].NumberOfOrders++
		}

		// check and remove empty price levels from slice
		if md.Buy[at].NumberOfOrders == 0 || md.Buy[at].Volume == 0 {
			md.Buy = append(md.Buy[:at], md.Buy[at+1:]...)
		}
		return
	}

	if orderInvalid {
		// Prevent filled orders that don't exist in a price/depth level from being added
		return
	}

	depthLevel := MarketDepthLevel{
		PriceLevel: types.PriceLevel{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1},
		orders:     map[string]uint64{order.Id: order.Remaining},
	}

	if at == -1 {
		// create a new MarketDepthLevel, non exist
		md.Buy = append(md.Buy, depthLevel)
	} else {
		// create new MarketDepthLevel for price at at appropriate position in slice
		md.Buy = append(md.Buy[:at], append([]MarketDepthLevel{depthLevel}, md.Buy[at:]...)...)
	}
}

// Called by Update to do the iteration over price levels and update the sell side of the market depth with
// order information. We now use a map of orderId => remaining to no longer need a DB lookup elsewhere.
func (md *marketDepth) updateSellSide(order types.Order) {
	var at = -1
	orderInvalid := md.isInvalid(order)

	// search through existing price/depth levels to find a position to insert
	found := false
	for idx, priceLevel := range md.Sell {
		if priceLevel.Price < order.Price {
			continue
		}
		if priceLevel.Price == order.Price {
			found = true
		}
		at = idx
		break
	}

	// check if the price/depth level was found for order price (to update existing total)
	if found {
		delta := uint64(0)

		// check if there's a previous order at this price level
		if existingRemaining, ok := md.Sell[at].orders[order.Id]; ok {
			// check if order is now fully filled or not trade-able status
			if orderInvalid {
				// order doesn't exist for price so remove from existing map
				md.Sell[at].Volume -= existingRemaining
				md.Sell[at].NumberOfOrders--
				delete(md.Sell[at].orders, order.Id)
			} else {
				delta = md.Sell[at].orders[order.Id] - order.Remaining
				md.Sell[at].orders[order.Id] = order.Remaining
				md.Sell[at].Volume -= delta
			}
		} else if !orderInvalid {
			md.Sell[at].orders[order.Id] = order.Remaining
			md.Sell[at].Volume += order.Remaining
			md.Sell[at].NumberOfOrders++
		}

		// check and remove empty price levels from slice
		if md.Sell[at].NumberOfOrders == 0 || md.Sell[at].Volume == 0 {
			md.Sell = append(md.Sell[:at], md.Sell[at+1:]...)
		}
		return
	}

	if orderInvalid {
		// Prevent filled orders that don't exist in a price/depth level from being added
		return
	}

	depthLevel := MarketDepthLevel{
		PriceLevel: types.PriceLevel{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1},
		orders:     map[string]uint64{order.Id: order.Remaining},
	}

	if at == -1 {
		// create a new MarketDepthLevel, non exist
		md.Sell = append(md.Sell, depthLevel)
	} else {
		// create new MarketDepthLevel for price at at appropriate position in slice
		md.Sell = append(md.Sell[:at], append([]MarketDepthLevel{depthLevel}, md.Sell[at:]...)...)
	}
}

// The buy side price levels (and additional information such as orders, remaining volumes) for the market.
func (md *marketDepth) BuySide() []MarketDepthLevel {
	return md.Buy
}

// The sell side price levels (and additional information such as orders, remaining volumes) for the market.
func (md *marketDepth) SellSide() []MarketDepthLevel {
	return md.Sell
}

// Helper to check for orders that have zero remaining, or a status such as cancelled etc.
// When calculating depth for a side they shouldn't be included (and removed if already exist).
// A fresh order with zero remaining will never be added to a price level.
func (md *marketDepth) isInvalid(order types.Order) bool {
	return order.Remaining == uint64(0) || order.Status == types.Order_Cancelled || order.Status == types.Order_Expired
}
