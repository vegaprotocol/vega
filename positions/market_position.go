package positions

import (
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
)

// MarketPosition represents the position of a party inside a market
type MarketPosition struct {
	// Actual volume
	size int64
	// Potential volume (orders not yet accepted/rejected)
	buy, sell int64

	partyID string
	price   uint64

	// volume weighted buy/sell prices
	vwBuyPrice, vwSellPrice uint64
}

func (p *MarketPosition) RegisterOrder(order *types.Order) {
	if order.Side == types.Side_SIDE_BUY {
		// calculate vwBuyPrice: total worth of orders divided by total size
		if buyVol := uint64(p.buy) + order.Remaining; buyVol != 0 {
			p.vwBuyPrice = (p.vwBuyPrice*uint64(p.buy) + order.Price*order.Remaining) / buyVol
		} else {
			p.vwBuyPrice = 0
		}
		p.buy += int64(order.Remaining)
		return
	}
	// calculate vwSellPrice: total worth of orders divided by total size
	if sellVol := uint64(p.sell) + order.Remaining; sellVol != 0 {
		p.vwSellPrice = (p.vwSellPrice*uint64(p.sell) + order.Price*order.Remaining) / sellVol
	} else {
		p.vwSellPrice = 0
	}
	p.sell += int64(order.Remaining)
}

func (p *MarketPosition) UnregisterOrder(order *types.Order) {
	if order.Side == types.Side_SIDE_BUY {
		// recalculate vwap
		vwap := p.vwBuyPrice*uint64(p.buy) - order.Price*order.Remaining
		p.buy -= int64(order.Remaining)
		if p.buy != 0 {
			p.vwBuyPrice = vwap / uint64(p.buy)
		} else {
			p.vwBuyPrice = 0
		}
		return
	}
	vwap := p.vwSellPrice*uint64(p.sell) - order.Price*order.Remaining
	p.sell -= int64(order.Remaining)
	if p.sell != 0 {
		p.vwSellPrice = vwap / uint64(p.sell)
	} else {
		p.vwSellPrice = 0
	}
}

// AmendOrder unregisters the original order and then registers the newly amended order
// this method is a quicker way of handling separate unregister+register pairs
func (p *MarketPosition) AmendOrder(originalOrder, newOrder *types.Order) {
	if originalOrder.Side == types.Side_SIDE_BUY {
		vwap := p.vwBuyPrice*uint64(p.buy) - originalOrder.Price*originalOrder.Remaining
		p.buy -= int64(originalOrder.Remaining)
		if p.buy != 0 {
			p.vwBuyPrice = vwap / uint64(p.buy)
		} else {
			p.vwBuyPrice = 0
		}
		p.buy += int64(newOrder.Remaining)
		return
	}
	vwap := p.vwSellPrice*uint64(p.sell) - originalOrder.Price*originalOrder.Remaining
	p.sell -= int64(originalOrder.Remaining)
	if p.sell != 0 {
		p.vwSellPrice = vwap / uint64(p.sell)
	} else {
		p.vwSellPrice = 0
	}
	p.sell += int64(newOrder.Remaining)
}

// String returns a string representation of a market
func (m MarketPosition) String() string {
	return fmt.Sprintf("size:%v, buy:%v, sell:%v, price:%v, partyID:%v",
		m.size, m.buy, m.sell, m.price, m.partyID)
}

// Buy will returns the potential buys for a given position
func (m MarketPosition) Buy() int64 {
	return m.buy
}

// Sell returns the potential sells for the position
func (m MarketPosition) Sell() int64 {
	return m.sell
}

// Size returns the current size of the position
func (m MarketPosition) Size() int64 {
	return m.size
}

// Party returns the party to which this positions is associated
func (m MarketPosition) Party() string {
	return m.partyID
}

// Price returns the current price for this position
func (m MarketPosition) Price() uint64 {
	return m.price
}

// VWBuy - get volume weighted buy price for unmatched buy orders
func (m MarketPosition) VWBuy() uint64 {
	return m.vwBuyPrice
}

// VWSell - get volume weighted sell price for unmatched sell orders
func (m MarketPosition) VWSell() uint64 {
	return m.vwSellPrice
}
