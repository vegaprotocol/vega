package datastore

import (
	"vega/msg"
)

// marketDepth stores the price levels on both buy side and sell side for a particular market.
type marketDepth struct {
	Name string
	Buy  []MarketDepthLevel
	Sell []MarketDepthLevel
}

type MarketDepthLevel struct {
	msg.PriceLevel              // price level details
	orders map[string]uint64    // map of order.Id => remaining value
}

func NewMarketDepth(name string) MarketDepth {
	return &marketDepth{Name: name}
}

type MarketDepth interface {
	Update(order msg.Order)
	BuySide() []MarketDepthLevel
	SellSide() []MarketDepthLevel
}

func (md *marketDepth) Update(order msg.Order) {
	if order.Side == msg.Side_Buy {
		md.updateBuySide(order)
	} else {
		md.updateSellSide(order)
	}
}

func (md *marketDepth) updateBuySide(order msg.Order) {
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
		PriceLevel: msg.PriceLevel{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1},
		orders:     map[string]uint64{ order.Id: order.Remaining },
	}

	if at == -1 {
		// create a new MarketDepthLevel, non exist
		md.Buy = append(md.Buy, depthLevel)
	} else {
		// create new MarketDepthLevel for price at at appropriate position in slice
		md.Buy = append(md.Buy[:at], append([]MarketDepthLevel{depthLevel}, md.Buy[at:]...)...)
	}
}

func (md *marketDepth) updateSellSide(order msg.Order) {
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
		PriceLevel: msg.PriceLevel{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1},
		orders:     map[string]uint64{ order.Id: order.Remaining },
	}

	if at == -1 {
		// create a new MarketDepthLevel, non exist
		md.Sell = append(md.Sell, depthLevel)
	} else {
		// create new MarketDepthLevel for price at at appropriate position in slice
		md.Sell = append(md.Sell[:at], append([]MarketDepthLevel{depthLevel}, md.Sell[at:]...)...)
	}
}

func (md *marketDepth) BuySide() []MarketDepthLevel {
	return md.Buy
}

func (md *marketDepth) SellSide() []MarketDepthLevel {
	return md.Sell
}

func (md *marketDepth) isInvalid(order msg.Order) bool {
	return order.Remaining == uint64(0) || order.Status == msg.Order_Cancelled || order.Status == msg.Order_Expired
}