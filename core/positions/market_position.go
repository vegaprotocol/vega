// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package positions

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

// MarketPosition represents the position of a party inside a market.
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

func NewMarketPosition(party string) *MarketPosition {
	return &MarketPosition{
		partyID:     party,
		price:       num.UintZero(),
		vwBuyPrice:  num.UintZero(),
		vwSellPrice: num.UintZero(),
	}
}

func (p MarketPosition) Clone() *MarketPosition {
	cpy := p
	cpy.price = p.price.Clone()
	cpy.vwBuyPrice = p.vwBuyPrice.Clone()
	cpy.vwSellPrice = p.vwSellPrice.Clone()
	return &cpy
}

func (p *MarketPosition) Closed() bool {
	// p.size can be negative
	// p.buy and p.sell can be only positive
	return p.size == 0 && p.buy+p.sell == 0
}

func (p *MarketPosition) SetParty(party string) { p.partyID = party }

func (p *MarketPosition) RegisterOrder(order *types.Order) {
	if order.Side == types.SideBuy {
		// calculate vwBuyPrice: total worth of orders divided by total size
		if buyVol := uint64(p.buy) + order.Remaining; buyVol != 0 {
			var a, b, c num.Uint
			// (p.vwBuyPrice*uint64(p.buy) + order.Price*order.Remaining) / buyVol
			a.Mul(p.vwBuyPrice, num.NewUint(uint64(p.buy)))
			b.Mul(order.Price, num.NewUint(order.Remaining))
			c.Add(&a, &b)
			p.vwBuyPrice.Div(&c, num.NewUint(buyVol))
		} else {
			p.vwBuyPrice.SetUint64(0)
		}
		p.buy += int64(order.Remaining)
		return
	}
	// calculate vwSellPrice: total worth of orders divided by total size
	if sellVol := uint64(p.sell) + order.Remaining; sellVol != 0 {
		var a, b, c num.Uint
		// (p.vwSellPrice*uint64(p.sell) + order.Price*order.Remaining) / sellVol
		a.Mul(p.vwSellPrice, num.NewUint(uint64(p.sell)))
		b.Mul(order.Price, num.NewUint(order.Remaining))
		c.Add(&a, &b)
		p.vwSellPrice.Div(&c, num.NewUint(sellVol))
	} else {
		p.vwSellPrice.SetUint64(0)
	}
	p.sell += int64(order.Remaining)
}

func (p *MarketPosition) UnregisterOrder(log *logging.Logger, order *types.Order) {
	if order.Side == types.SideBuy {
		if uint64(p.buy) < order.Remaining {
			log.Panic("cannot unregister order with remaining > potential buy",
				logging.Order(*order),
				logging.Int64("potential-buy", p.buy))
		}
		// recalculate vwap
		var a, b, vwap num.Uint
		// p.vwBuyPrice*uint64(p.buy) - order.Price*order.Remaining
		a.Mul(p.vwBuyPrice, num.NewUint(uint64(p.buy)))
		b.Mul(order.Price, num.NewUint(order.Remaining))
		vwap.Sub(&a, &b)
		p.buy -= int64(order.Remaining)
		if p.buy != 0 {
			p.vwBuyPrice.Div(&vwap, num.NewUint(uint64(p.buy)))
		} else {
			p.vwBuyPrice.SetUint64(0)
		}
		return
	}

	if uint64(p.sell) < order.Remaining {
		log.Panic("cannot unregister order with remaining > potential sell",
			logging.Order(*order),
			logging.Int64("potential-sell", p.sell))
	}

	var a, b, vwap num.Uint
	// p.vwSellPrice*uint64(p.sell) - order.Price*order.Remaining
	a.Mul(p.vwSellPrice, num.NewUint(uint64(p.sell)))
	b.Mul(order.Price, num.NewUint(order.Remaining))
	vwap.Sub(&a, &b)
	p.sell -= int64(order.Remaining)
	if p.sell != 0 {
		p.vwSellPrice.Div(&vwap, num.NewUint(uint64(p.sell)))
	} else {
		p.vwSellPrice.SetUint64(0)
	}
}

// AmendOrder unregisters the original order and then registers the newly amended order
// this method is a quicker way of handling separate unregister+register pairs.
func (p *MarketPosition) AmendOrder(log *logging.Logger, originalOrder, newOrder *types.Order) {
	switch originalOrder.Side {
	case types.SideBuy:
		if uint64(p.buy) < originalOrder.Remaining {
			log.Panic("cannot amend order with remaining > potential buy",
				logging.Order(*originalOrder),
				logging.Int64("potential-buy", p.buy))
		}
	case types.SideSell:
		if uint64(p.sell) < originalOrder.Remaining {
			log.Panic("cannot amend order with remaining > potential sell",
				logging.Order(*originalOrder),
				logging.Int64("potential-sell", p.sell))
		}
	}

	p.UnregisterOrder(log, originalOrder)
	p.RegisterOrder(newOrder)
}

// String returns a string representation of a market.
func (p MarketPosition) String() string {
	return fmt.Sprintf("size:%v, buy:%v, sell:%v, price:%v, partyID:%v",
		p.size, p.buy, p.sell, p.price, p.partyID)
}

// Buy will returns the potential buys for a given position.
func (p MarketPosition) Buy() int64 {
	return p.buy
}

// Sell returns the potential sells for the position.
func (p MarketPosition) Sell() int64 {
	return p.sell
}

// Size returns the current size of the position.
func (p MarketPosition) Size() int64 {
	return p.size
}

// Party returns the party to which this positions is associated.
func (p MarketPosition) Party() string {
	return p.partyID
}

// Price returns the current price for this position.
func (p MarketPosition) Price() *num.Uint {
	if p.price != nil {
		return p.price.Clone()
	}
	return num.UintZero()
}

// VWBuy - get volume weighted buy price for unmatched buy orders.
func (p MarketPosition) VWBuy() *num.Uint {
	if p.vwBuyPrice != nil {
		return p.vwBuyPrice.Clone()
	}
	return num.UintZero()
}

// VWSell - get volume weighted sell price for unmatched sell orders.
func (p MarketPosition) VWSell() *num.Uint {
	if p.vwSellPrice != nil {
		return p.vwSellPrice.Clone()
	}
	return num.UintZero()
}

func (p MarketPosition) OrderReducesExposure(ord *types.Order) bool {
	if ord == nil || p.Size() == 0 || ord.PeggedOrder != nil {
		return false
	}
	// long position and short order
	if p.Size() > 0 && ord.Side == types.SideSell {
		// market order reduces exposure and doesn't flip position to the other side
		if p.Size()-int64(ord.Remaining) >= 0 && ord.Type == types.OrderTypeMarket {
			return true
		}
		// sum of all short limit orders wouldn't flip the position if filled (ord already included in pos)
		if p.Size()-p.Sell() >= 0 && ord.Type == types.OrderTypeLimit {
			return true
		}
	}
	// short position and long order
	if p.Size() < 0 && ord.Side == types.SideBuy {
		// market order reduces exposure and doesn't flip position to the other side
		if p.Size()+int64(ord.Remaining) <= 0 && ord.Type == types.OrderTypeMarket {
			return true
		}
		// sum of all long limit orders wouldn't flip the position if filled (ord already included in pos)
		if p.Size()+p.Buy() <= 0 && ord.Type == types.OrderTypeLimit {
			return true
		}
	}
	return false
}

// OrderReducesOnlyExposure returns true if the order reduce the position and the extra size if it was to flip the position side.
func (p MarketPosition) OrderReducesOnlyExposure(ord *types.Order) (reduce bool, extraSize uint64) {
	// if already closed, or increasing position, we shortcut
	if p.Size() == 0 || (p.Size() < 0 && ord.Side == types.SideSell) || (p.Size() > 0 && ord.Side == types.SideBuy) {
		return false, 0
	}

	size := p.Size()
	if size < 0 {
		size = -size
	}
	if extraSizeI := size - int64(ord.Remaining); extraSizeI < 0 {
		return true, uint64(-extraSizeI)
	}
	return true, 0
}
