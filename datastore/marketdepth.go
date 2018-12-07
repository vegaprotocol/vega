package datastore

import (
	"vega/msg"
)

type MarketDepth struct {
	Name string
	Buy  []*msg.PriceLevel
	Sell []*msg.PriceLevel
}

type MarketDepthManager interface {
	Add(order *msg.Order)
	DecreaseByTradedVolume(order *msg.Order, tradedVolume uint64)
	getBuySide() []*msg.PriceLevel
	getSellSide() []*msg.PriceLevel
}

func NewMarketDepthUpdaterGetter() MarketDepthManager {
	return &MarketDepth{}
}

// recalculate cumulative volume only once when fetching the MarketDepth
func (md *MarketDepth) addToBuySide(order *msg.Order) {
	var at = -1

	for idx, priceLevel := range md.Buy {
		if priceLevel.Price > order.Price {
			continue
		}

		if priceLevel.Price == order.Price {
			// add to price level
			md.Buy[idx].Volume += order.Remaining
			md.Buy[idx].NumberOfOrders++
			// updated - job done
			return
		}

		at = idx
		break
	}

	if at == -1 {
		// reached the end and not found, append at the end
		md.Buy = append(md.Buy, &msg.PriceLevel{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1})
		return
	}
	// found insert at
	md.Buy = append(md.Buy[:at], append([]*msg.PriceLevel{{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1}}, md.Buy[at:]...)...)
}

func (md *MarketDepth) addToSellSide(order *msg.Order) {
	var at = -1

	for idx, priceLevel := range md.Sell {
		if priceLevel.Price < order.Price {
			continue
		}

		if priceLevel.Price == order.Price {
			// add to price level
			md.Sell[idx].Volume += order.Remaining
			md.Sell[idx].NumberOfOrders++
			// updated - job done
			return
		}

		at = idx
		break
	}

	if at == -1 {
		md.Sell = append(md.Sell, &msg.PriceLevel{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1})
		return
	}
	// found insert at
	md.Sell = append(md.Sell[:at], append([]*msg.PriceLevel{{Price: order.Price, Volume: order.Remaining, NumberOfOrders: 1}}, md.Sell[at:]...)...)
}

func (md *MarketDepth) Add(order *msg.Order) {
	if order.Side == msg.Side_Buy {
		md.addToBuySide(order)
	}
	if order.Side == msg.Side_Sell {
		md.addToSellSide(order)
	}
}

func (md *MarketDepth) DecreaseByTradedVolume(order *msg.Order, tradedVolume uint64) {
	if order.Side == msg.Side_Buy {
		for idx, priceLevel := range md.Buy {
			if priceLevel.Price > order.Price {
				continue
			}

			if priceLevel.Price == order.Price {

				// Smart trick to check overflow, remove level which goes negative
				if md.Buy[idx].Volume - order.Remaining > md.Buy[idx].Volume {
					copy(md.Buy[idx:], md.Buy[idx+1:])
					md.Buy = md.Buy[:len(md.Buy)-1]
					return
				}

				// update price level
				md.Buy[idx].Volume -= tradedVolume
				if order.Remaining == uint64(0) || order.Status == msg.Order_Cancelled || order.Status == msg.Order_Expired {
					md.Buy[idx].NumberOfOrders--
					md.Buy[idx].Volume -= order.Remaining
				}
				// updated - job done

				if md.Buy[idx].NumberOfOrders == 0 {
					copy(md.Buy[idx:], md.Buy[idx+1:])
					md.Buy = md.Buy[:len(md.Buy)-1]
				}
				return
			}
		}
		// not found
		return
	}

	if order.Side == msg.Side_Sell {
		for idx, priceLevel := range md.Sell {
			if priceLevel.Price < order.Price {
				continue
			}

			if priceLevel.Price == order.Price {

				// Smart trick to check overflow, remove level which goes negative
				if md.Sell[idx].Volume - order.Remaining > md.Sell[idx].Volume {
					copy(md.Sell[idx:], md.Sell[idx+1:])
					md.Sell = md.Sell[:len(md.Sell)-1]
					return
				}

				// update price level
				md.Sell[idx].Volume -= tradedVolume
				if order.Remaining == uint64(0) || order.Status == msg.Order_Cancelled || order.Status == msg.Order_Expired {
					md.Sell[idx].NumberOfOrders--
					md.Sell[idx].Volume -= order.Remaining
				}
				// updated - job done

				// safeguard -  negative volume shouldn't happen but if volume for gets negative remove price level
				if md.Sell[idx].NumberOfOrders == 0 || md.Sell[idx].Volume <= 0 {
					copy(md.Sell[idx:], md.Sell[idx+1:])
					md.Sell = md.Sell[:len(md.Sell)-1]
				}
				return
			}
		}
		// not found
		return
	}
}

func (md *MarketDepth) getBuySide() []*msg.PriceLevel {
	return md.Buy
}

func (md *MarketDepth) getSellSide() []*msg.PriceLevel {
	return md.Sell
}
