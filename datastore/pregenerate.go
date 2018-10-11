package datastore

import (
	"vega/msg"
	"fmt"
)

type MarketDepth struct {
	Name string
	Buy []*msg.PriceLevel
	Sell []*msg.PriceLevel
}

func (md *MarketDepth) updateWithRemaining(order *Order) {
	if order.Side == msg.Side_Buy {

		if len(md.Buy) == 0 {
			md.Buy = []*msg.PriceLevel{{Price: order.Price, Volume: order.Remaining, NumberOfOrders:1}}
			fmt.Printf("placed\n")
			fmt.Printf("placed %d\n", len(md.Buy))
			return
		}

		for idx, priceLevel := range md.Buy {
			if priceLevel.Price > order.Price {
				continue
			}

			if priceLevel.Price == order.Price {
				// update price level
				md.Buy[idx].Volume += order.Remaining
				md.Buy[idx].NumberOfOrders++
				break
				// recalculate cumulative volume only once when fetch
			}

			if priceLevel.Price <= order.Price {
				// price level does not exist - insert here
				md.Buy = append(md.Buy[:idx], append([]*msg.PriceLevel{{Price: order.Price, Volume: order.Remaining, NumberOfOrders:1}}, md.Buy[idx:]...)...)
				break
			}
		}
		return
	}

	if order.Side == msg.Side_Sell {

		if len(md.Sell) == 0 {
			md.Sell = []*msg.PriceLevel{{Price: order.Price, Volume: order.Remaining, NumberOfOrders:1}}
			return
		}

		for idx, priceLevel := range md.Sell {
			if priceLevel.Price < order.Price {
				continue
			}

			if priceLevel.Price == order.Price {
				// update price level
				md.Sell[idx].Volume += order.Remaining
				md.Sell[idx].NumberOfOrders++
				break
				// recalculate cumulative volume only once when fetch
			}

			if priceLevel.Price >= order.Price {
				// price level does not exist - insert here
				md.Sell = append(md.Sell[:idx], append([]*msg.PriceLevel{{Price: order.Price, Volume: order.Remaining, NumberOfOrders:1}}, md.Sell[idx:]...)...)
				break
			}
		}
		return
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
				break
				// recalculate cumulative volume only once when fetch
			}
		}
		return
	}

	if order.Side == msg.Side_Sell {
		for idx, priceLevel := range md.Sell {
			if priceLevel.Price < order.Price {
				continue
			}

			if priceLevel.Price == order.Price {
				// update price level
				md.Buy[idx].Volume -= remainingDelta
				break
				// recalculate cumulative volume only once when fetch
			}
		}
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
				if md.Buy[idx].NumberOfOrders == 0 {
					copy(md.Buy[idx:], md.Buy[idx+1:])
					md.Buy = md.Buy[:len(md.Buy)-1]
				}
				break
				// recalculate cumulative volume only once when fetch
			}
		}
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
				md.Buy[idx].Volume -= order.Remaining
				if md.Sell[idx].NumberOfOrders == 0 {
					copy(md.Sell[idx:], md.Sell[idx+1:])
					md.Sell = md.Sell[:len(md.Sell)-1]
				}
				break
				// recalculate cumulative volume only once when fetch
			}
		}
		return
	}
}