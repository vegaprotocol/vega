package datastore

import (
	"vega/msg"
)

// MarketDepth stores the price levels on both buy side and sell side for a particular market.
type Depth struct {
	Name string
	Buy  []DepthLevel
	Sell []DepthLevel
}

type DepthLevel struct {
	msg.PriceLevel              // price level details
	orders map[string]uint64    // map of order.Id => remaining value
}

func NewDepth() DepthManager {
	return &Depth{}
}

type DepthManager interface {
	Update(order msg.Order)
	BuySide() []DepthLevel
	SellSide() []DepthLevel
}

func (md *Depth) Update(order msg.Order) {
	if order.Side == msg.Side_Buy {
		md.updateBuySide(order)
	} else {
		md.updateSellSide(order)
	}
}

func (md *Depth) updateBuySide(order msg.Order) {
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
		// check if there's a previous order at this price level
		if _, ok := md.Buy[at].orders[order.Id]; ok {
			// check if order is now fully filled or not trade-able status
			if orderInvalid {
				// order doesn't exist for price so remove from existing map
				delete(md.Buy[at].orders, order.Id)
				//fmt.Println("order deleted", order.Side.String(), order.Id, len(md.Buy[at].orders))
			} else {
				md.Buy[at].orders[order.Id] = order.Remaining
			}
		} else if !orderInvalid {
			md.Buy[at].orders[order.Id] = order.Remaining
		}

		// recalculate totals
		vol, total := uint64(0), uint64(0)
		for _, v := range md.Buy[at].orders {
			vol += v
			total++
		}
		md.Buy[at].Volume = vol
		md.Buy[at].NumberOfOrders = total

		// check and remove empty price levels from slice
		if md.Buy[at].NumberOfOrders == 0 || md.Buy[at].Volume == 0 {
			//fmt.Println("removing price level ", order.Side.String(), md.Buy[at].Price, md.Buy[at].Volume, md.Buy[at].NumberOfOrders)
			md.Buy = append(md.Buy[:at], md.Buy[at+1:]...)
		}
		return
	}

	if orderInvalid {
		return
	}

	depthLevel := DepthLevel{
		PriceLevel: msg.PriceLevel{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1},
		orders:     map[string]uint64{ order.Id: order.Remaining },
	}

	if at == -1 {
		// create a new DepthLevel, non exist
		md.Buy = append(md.Buy, depthLevel)
	} else {
		// create new DepthLevel for price at at appropriate position in slice
		md.Buy = append(md.Buy[:at], append([]DepthLevel{depthLevel}, md.Buy[at:]...)...)
	}
}

func (md *Depth) updateSellSide(order msg.Order) {
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

		// check if there's a previous order at this price level
		if _, ok := md.Sell[at].orders[order.Id]; ok {
			// check if order is now fully filled or not trade-able status
			if orderInvalid {
				// order doesn't exist for price so remove from existing map
				delete(md.Sell[at].orders, order.Id)
				//fmt.Println("order deleted", order.Side.String(), order.Id, len(md.Sell[at].orders))
			} else {
				md.Sell[at].orders[order.Id] = order.Remaining
			}
		} else if !orderInvalid {
			md.Sell[at].orders[order.Id] = order.Remaining
		}

		// recalculate totals
		vol, total := uint64(0), uint64(0)
		for _, v := range md.Sell[at].orders {
			vol += v
			total++
		}
		md.Sell[at].Volume = vol
		md.Sell[at].NumberOfOrders = total

		// check and remove empty price levels from slice
		if md.Sell[at].NumberOfOrders == 0 || md.Sell[at].Volume == 0 {
			//fmt.Println("removing price level ", order.Side.String(), md.Sell[at].Price, md.Sell[at].Volume, md.Sell[at].NumberOfOrders)
			md.Sell = append(md.Sell[:at], md.Sell[at+1:]...)
		}
		return
	}

	if orderInvalid {
		return
	}

	depthLevel := DepthLevel{
		PriceLevel: msg.PriceLevel{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1},
		orders:     map[string]uint64{ order.Id: order.Remaining },
	}

	if at == -1 {
		// create a new DepthLevel, non exist 
		md.Sell = append(md.Sell, depthLevel)
	} else {
		// create new DepthLevel for price at at appropriate position in slice
		md.Sell = append(md.Sell[:at], append([]DepthLevel{depthLevel}, md.Sell[at:]...)...)
	}
}

func (md *Depth) BuySide() []DepthLevel {
	return md.Buy
}

func (md *Depth) SellSide() []DepthLevel {
	return md.Sell
}

func (md *Depth) isInvalid(order msg.Order) bool {
	return order.Remaining == uint64(0) || order.Status == msg.Order_Cancelled || order.Status == msg.Order_Expired
}