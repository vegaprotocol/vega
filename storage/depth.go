package storage

import (
	types "code.vegaprotocol.io/vega/proto"
)

// Depth stores the price levels on both buy side and sell side for a particular market.
type Depth struct {
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
func NewMarketDepth(name string) *Depth {
	return &Depth{Name: name}
}

// Update the market depth with the given order information. If the order already exists at a price level
// it will be updated. Note: The total cumulative volume for the market depth is calculated elsewhere.
func (d *Depth) Update(order types.Order) {
	if order.TimeInForce != types.Order_TIF_IOC && order.TimeInForce != types.Order_TIF_FOK && order.Status != types.Order_STATUS_REJECTED && order.Type != types.Order_TYPE_NETWORK {
		if order.Side == types.Side_SIDE_BUY {
			d.updateBuySide(order)
		} else {
			d.updateSellSide(order)
		}
	}
}

// Called by Update to do the iteration over price levels and update the buy side of the market depth with
// order information. We now use a map of orderId => remaining to no longer need a DB lookup elsewhere.
func (d *Depth) updateBuySide(order types.Order) {
	var at = -1
	orderInvalid := d.isInvalid(order)

	// search through existing price/depth levels to find a position to insert
	found := false
	for idx, priceLevel := range d.Buy {
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
		// check if there's a previous order at this price level
		if existingRemaining, ok := d.Buy[at].orders[order.Id]; ok {
			// check if order is now fully filled or not trade-able status
			if orderInvalid {
				// order doesn't exist for price so remove from existing map
				d.Buy[at].Volume -= existingRemaining
				d.Buy[at].NumberOfOrders--
				delete(d.Buy[at].orders, order.Id)
			} else {
				delta := d.Buy[at].orders[order.Id] - order.Remaining
				d.Buy[at].orders[order.Id] = order.Remaining
				d.Buy[at].Volume -= delta
			}
		} else if !orderInvalid {
			d.Buy[at].orders[order.Id] = order.Remaining
			d.Buy[at].Volume += order.Remaining
			d.Buy[at].NumberOfOrders++
		}

		// check and remove empty price levels from slice
		if d.Buy[at].NumberOfOrders == 0 || d.Buy[at].Volume == 0 {
			d.Buy = append(d.Buy[:at], d.Buy[at+1:]...)
		}
		return
	}

	if orderInvalid {
		// Prevent filled orders that don't exist in a price/depth level from being added
		return
	}

	depthLevel := MarketDepthLevel{
		PriceLevel: types.PriceLevel{
			Price:          order.Price,
			Volume:         order.Remaining,
			NumberOfOrders: 1,
		},
		orders: map[string]uint64{
			order.Id: order.Remaining,
		},
	}

	if at == -1 {
		// create a new MarketDepthLevel, non exist
		d.Buy = append(d.Buy, depthLevel)
		return
	}
	// create new MarketDepthLevel for price at at appropriate position in slice
	d.Buy = append(d.Buy[:at], append([]MarketDepthLevel{depthLevel}, d.Buy[at:]...)...)
}

// Called by Update to do the iteration over price levels and update the sell side of the market depth with
// order information. We now use a map of orderId => remaining to no longer need a DB lookup elsewhere.
func (d *Depth) updateSellSide(order types.Order) {
	var at = -1
	orderInvalid := d.isInvalid(order)

	// search through existing price/depth levels to find a position to insert
	found := false
	for idx, priceLevel := range d.Sell {
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
		// check if there's a previous order at this price level
		if existingRemaining, ok := d.Sell[at].orders[order.Id]; ok {
			// check if order is now fully filled or not trade-able status
			if orderInvalid {
				// order doesn't exist for price so remove from existing map
				d.Sell[at].Volume -= existingRemaining
				d.Sell[at].NumberOfOrders--
				delete(d.Sell[at].orders, order.Id)
			} else {
				delta := d.Sell[at].orders[order.Id] - order.Remaining
				d.Sell[at].orders[order.Id] = order.Remaining
				d.Sell[at].Volume -= delta
			}
		} else if !orderInvalid {
			d.Sell[at].orders[order.Id] = order.Remaining
			d.Sell[at].Volume += order.Remaining
			d.Sell[at].NumberOfOrders++
		}

		// check and remove empty price levels from slice
		if d.Sell[at].NumberOfOrders == 0 || d.Sell[at].Volume == 0 {
			d.Sell = append(d.Sell[:at], d.Sell[at+1:]...)
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
		d.Sell = append(d.Sell, depthLevel)
	} else {
		// create new MarketDepthLevel for price at at appropriate position in slice
		d.Sell = append(d.Sell[:at], append([]MarketDepthLevel{depthLevel}, d.Sell[at:]...)...)
	}
}

// BuySide The buy side price levels (and additional information such as orders,
// remaining volumes) for the market.
func (d *Depth) BuySide(limit uint64) []MarketDepthLevel {
	if limit == 0 || limit > uint64(len(d.Buy)) {
		return d.Buy
	}
	return d.Buy[:limit]
}

// SellSide The sell side price levels (and additional information such as
// orders, remaining volumes) for the market.
func (d *Depth) SellSide(limit uint64) []MarketDepthLevel {
	if limit == 0 || limit > uint64(len(d.Sell)) {
		return d.Sell
	}
	return d.Sell[:limit]
}

// Helper to check for orders that have zero remaining, or a status such as cancelled etc.
// When calculating depth for a side they shouldn't be included (and removed if already exist).
// A fresh order with zero remaining will never be added to a price level.
func (d *Depth) isInvalid(order types.Order) bool {
	return order.Remaining == uint64(0) || order.Status == types.Order_STATUS_CANCELLED || order.Status == types.Order_STATUS_EXPIRED || order.Status == types.Order_STATUS_PARTIALLY_FILLED || order.Status == types.Order_STATUS_STOPPED
}
