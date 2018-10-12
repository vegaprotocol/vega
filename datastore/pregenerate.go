package datastore

import (
	"vega/msg"
)

type MarketDepth struct {
	Name string
	Buy []*msg.PriceLevel
	Sell []*msg.PriceLevel
}

// recalculate cumulative volume only once when fetching the MarketDepth

func (md *MarketDepth) updateWithRemainingBuySide(order *Order) {
	var at = -1

	for idx, priceLevel := range md.Buy {
		if priceLevel.Price > order.Price {
			continue
		}

		if priceLevel.Price == order.Price {
			// update price level
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
	md.Buy = append(md.Buy[:at], append([]*msg.PriceLevel{{Price: order.Price, Volume: order.Remaining, NumberOfOrders:1}}, md.Buy[at:]...)...)
}

func (md *MarketDepth) updateWithRemainingSellSide(order *Order) {
	var at = -1

	for idx, priceLevel := range md.Sell {
		if priceLevel.Price < order.Price {
			continue
		}

		if priceLevel.Price == order.Price {
			// update price level
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
	md.Sell = append(md.Sell[:at], append([]*msg.PriceLevel{{Price: order.Price, Volume: order.Remaining, NumberOfOrders:1}}, md.Sell[at:]...)...)
}

func (md *MarketDepth) updateWithRemaining(order *Order) {
	if order.Side == msg.Side_Buy {
		md.updateWithRemainingBuySide(order)
	}
	if order.Side == msg.Side_Sell {
		md.updateWithRemainingSellSide(order)
	}
}

func (md *MarketDepth) updateWithRemainingDelta(order *Order, remainingDelta uint64) {
	if order.Side == msg.Side_Buy {
		for idx, priceLevel := range md.Buy {
			if priceLevel.Price > order.Price {
				continue
			}

			if priceLevel.Price == order.Price {
				// update price level
				md.Buy[idx].Volume -= remainingDelta
				// updated - job done
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
				// update price level
				md.Sell[idx].Volume -= remainingDelta
				// updated - job done
				return
			}
		}
		// not found
		return
	}
}

func (md *MarketDepth) removeWithRemaining(order *Order) {
	if order.Side == msg.Side_Buy {
		for idx, priceLevel := range md.Buy {
			if priceLevel.Price > order.Price {
				continue
			}

			if priceLevel.Price == order.Price {
				// update price level
				md.Buy[idx].NumberOfOrders--
				md.Buy[idx].Volume -= order.Remaining

				// remove empty price level
				if md.Buy[idx].NumberOfOrders == 0 {
					copy(md.Buy[idx:], md.Buy[idx+1:])
					md.Buy = md.Buy[:len(md.Buy)-1]
				}
				// updated - job done
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
				// update price level
				md.Sell[idx].NumberOfOrders--
				md.Sell[idx].Volume -= order.Remaining

				// remove empty price level
				if md.Sell[idx].NumberOfOrders == 0 {
					copy(md.Sell[idx:], md.Sell[idx+1:])
					md.Sell = md.Sell[:len(md.Sell)-1]
				}
				// updated - job done
				return
			}
		}
		// not found
		return
	}
}