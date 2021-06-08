package positions

import (
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

// MarketPosition represents the position of a party inside a market
type MarketPosition struct {
	// Actual volume
	size int64
	// Potential volume (orders not yet accepted/rejected)
	buy, sell int64

	partyID string
	price   *num.Uint

	// volume weighted buy/sell prices
	vwBuyPrice, vwSellPrice *num.Uint
}

func (p *MarketPosition) SetParty(party string) { p.partyID = party }

func (p *MarketPosition) RegisterOrder(order *types.Order) {
	if order.Side == types.Side_SIDE_BUY {
		// calculate vwBuyPrice: total worth of orders divided by total size
		if buyVol := uint64(p.buy) + order.Remaining; buyVol != 0 {
			var (
				a num.Uint
				b num.Uint
				c num.Uint
			)
			// (p.vwBuyPrice*uint64(p.buy) + order.Price*order.Remaining) / buyVol
			a.Mul(p.vwBuyPrice, num.NewUint(uint64(p.buy)))
			b.Mul(order.Price, num.NewUint(order.Remaining))
			c.Add(&a, &b)
			p.vwBuyPrice.Div(&c, num.NewUint(buyVol))
		} else {
			p.vwBuyPrice = num.NewUint(0)
		}
		p.buy += int64(order.Remaining)
		return
	}
	// calculate vwSellPrice: total worth of orders divided by total size
	if sellVol := uint64(p.sell) + order.Remaining; sellVol != 0 {
		var (
			a num.Uint
			b num.Uint
			c num.Uint
		)
		// (p.vwSellPrice*uint64(p.sell) + order.Price*order.Remaining) / sellVol
		a.Mul(p.vwSellPrice, num.NewUint(uint64(p.sell)))
		b.Mul(order.Price, num.NewUint(order.Remaining))
		c.Add(&a, &b)
		p.vwSellPrice.Div(&c, num.NewUint(sellVol))
	} else {
		p.vwSellPrice = num.NewUint(0)
	}
	p.sell += int64(order.Remaining)
}

func (p *MarketPosition) UnregisterOrder(log *logging.Logger, order *types.Order) {
	if order.Side == types.Side_SIDE_BUY {
		if uint64(p.buy) < order.Remaining {
			log.Panic("cannot unregister order with remaining > potential buy",
				logging.Order(*order),
				logging.Int64("potential-buy", p.buy))
		}
		// recalculate vwap
		var (
			a num.Uint
			b num.Uint
			c num.Uint
		)
		// p.vwBuyPrice*uint64(p.buy) - order.Price*order.Remaining
		a.Mul(p.vwBuyPrice, num.NewUint(uint64(p.buy)))
		b.Mul(order.Price, num.NewUint(order.Remaining))
		c.Sub(&a, &b)
		vwap := c.Uint64()
		p.buy -= int64(order.Remaining)
		if p.buy != 0 {
			p.vwBuyPrice = num.NewUint(vwap / uint64(p.buy))
		} else {
			p.vwBuyPrice = num.NewUint(0)
		}
		return
	}

	if uint64(p.sell) < order.Remaining {
		log.Panic("cannot unregister order with remaining > potential sell",
			logging.Order(*order),
			logging.Int64("potential-sell", p.sell))
	}

	var (
		a num.Uint
		b num.Uint
		c num.Uint
	)
	// p.vwSellPrice*uint64(p.sell) - order.Price*order.Remaining
	a.Mul(p.vwSellPrice, num.NewUint(uint64(p.sell)))
	b.Mul(order.Price, num.NewUint(order.Remaining))
	c.Sub(&a, &b)
	vwap := c.Uint64()
	p.sell -= int64(order.Remaining)
	if p.sell != 0 {
		p.vwSellPrice = num.NewUint(vwap / uint64(p.sell))
	} else {
		p.vwSellPrice = num.NewUint(0)
	}
}

// AmendOrder unregisters the original order and then registers the newly amended order
// this method is a quicker way of handling separate unregister+register pairs
func (p *MarketPosition) AmendOrder(log *logging.Logger, originalOrder, newOrder *types.Order) {
	if originalOrder.Side == types.Side_SIDE_BUY {
		if uint64(p.buy) < originalOrder.Remaining {
			log.Panic("cannot amend order with remaining > potential buy",
				logging.Order(*originalOrder),
				logging.Int64("potential-buy", p.buy))
		}

		var (
			a num.Uint
			b num.Uint
			c num.Uint
		)
		// p.vwBuyPrice*uint64(p.buy) - originalOrder.Price*originalOrder.Remaining
		a.Mul(p.vwBuyPrice, num.NewUint(uint64(p.buy)))
		b.Mul(originalOrder.Price, num.NewUint(originalOrder.Remaining))
		c.Sub(&a, &b)
		vwap := c.Uint64()
		p.buy -= int64(originalOrder.Remaining)
		if p.buy != 0 {
			p.vwBuyPrice = num.NewUint(vwap / uint64(p.buy))
		} else {
			p.vwBuyPrice = num.NewUint(0)
		}
		p.buy += int64(newOrder.Remaining)
		return
	}

	if uint64(p.sell) < originalOrder.Remaining {
		log.Panic("cannot amend order with remaining > potential sell",
			logging.Order(*originalOrder),
			logging.Int64("potential-sell", p.sell))
	}

	var (
		a num.Uint
		b num.Uint
		c num.Uint
	)
	// p.vwSellPrice*uint64(p.sell) - originalOrder.Price*originalOrder.Remaining
	a.Mul(p.vwSellPrice, num.NewUint(uint64(p.sell)))
	b.Mul(originalOrder.Price, num.NewUint(originalOrder.Remaining))
	c.Sub(&a, &b)
	vwap := c.Uint64()
	p.sell -= int64(originalOrder.Remaining)
	if p.sell != 0 {
		p.vwSellPrice = num.NewUint(vwap / uint64(p.sell))
	} else {
		p.vwSellPrice = num.NewUint(0)
	}
	p.sell += int64(newOrder.Remaining)
}

// String returns a string representation of a market
func (p MarketPosition) String() string {
	return fmt.Sprintf("size:%v, buy:%v, sell:%v, price:%v, partyID:%v",
		p.size, p.buy, p.sell, p.price, p.partyID)
}

// Buy will returns the potential buys for a given position
func (p MarketPosition) Buy() int64 {
	return p.buy
}

// Sell returns the potential sells for the position
func (p MarketPosition) Sell() int64 {
	return p.sell
}

// Size returns the current size of the position
func (p MarketPosition) Size() int64 {
	return p.size
}

// Party returns the party to which this positions is associated
func (p MarketPosition) Party() string {
	return p.partyID
}

// Price returns the current price for this position
func (p MarketPosition) Price() *num.Uint {
	return p.price.Clone()
}

// VWBuy - get volume weighted buy price for unmatched buy orders
func (p MarketPosition) VWBuy() *num.Uint {
	return p.vwBuyPrice.Clone()
}

// VWSell - get volume weighted sell price for unmatched sell orders
func (p MarketPosition) VWSell() *num.Uint {
	return p.vwSellPrice.Clone()
}
