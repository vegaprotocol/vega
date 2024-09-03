// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package amm

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

type shapeMaker struct {
	log   *logging.Logger
	idgen *idgeneration.IDGenerator

	pool      *Pool     // the AMM we are expanding into orders
	pos       int64     // the AMM's current position
	fairPrice *num.Uint // the AMM's fair-price

	step    *num.Uint // price step we will be taking as we walk over the curves
	approx  bool      // whether we are taking approximate steps
	oneTick *num.Uint // one price level tick which may be bigger than one given the markets price factor

	buys  []*types.Order // buy orders are added here as we calculate them
	sells []*types.Order // sell orders are added here as we calculate them
	side  types.Side     // the side the *next* calculated order will be on

	from *num.Uint // the adjusted start region i.e the input region capped to the AMM's bounds
	to   *num.Uint // the adjusted end region
}

func newShapeMaker(log *logging.Logger, p *Pool, from, to *num.Uint, idgen *idgeneration.IDGenerator) *shapeMaker {
	buys := make([]*types.Order, 0, p.maxCalculationLevels.Uint64())
	sells := make([]*types.Order, 0, p.maxCalculationLevels.Uint64())

	return &shapeMaker{
		log:       log,
		pool:      p,
		pos:       p.getPosition(),
		fairPrice: p.fairPrice(),
		buys:      buys,
		sells:     sells,
		from:      from.Clone(),
		to:        to.Clone(),
		side:      types.SideBuy,
		oneTick:   p.oneTick.Clone(),
		idgen:     idgen,
	}
}

// addOrder creates an order with the given details and adds it to the relevant slice based on its side.
func (e *shapeMaker) addOrder(volume uint64, price *num.Uint, side types.Side) {
	if e.log.IsDebug() {
		e.log.Debug("creating shape order",
			logging.String("price", price.String()),
			logging.String("side", side.String()),
			logging.Uint64("volume", volume),
			logging.String("amm-party", e.pool.AMMParty),
		)
	}

	e.appendOrder(e.pool.makeOrder(volume, price, side, e.idgen))
}

// appendOrder takes the concrete order and appends it to the relevant slice based on its side.
func (sm *shapeMaker) appendOrder(o *types.Order) {
	if o.Side == types.SideBuy {
		sm.buys = append(sm.buys, o)
		return
	}
	sm.sells = append(sm.sells, o)
}

// makeBoundaryOrder creates an accurrate order for the given one-tick interval which will exist at the edges
// of the adjusted expansion region.
func (sm *shapeMaker) makeBoundaryOrder(st, nd *num.Uint) *types.Order {
	// lets do the starting boundary order
	cu := sm.pool.lower
	if st.GTE(sm.pool.lower.high) {
		cu = sm.pool.upper
	}

	volume := num.DeltaV(
		cu.positionAtPrice(sm.pool.sqrt, st),
		cu.positionAtPrice(sm.pool.sqrt, nd),
	)

	if st.GTE(sm.fairPrice) {
		return sm.pool.makeOrder(uint64(volume), nd, types.SideSell, sm.idgen)
	}

	return sm.pool.makeOrder(uint64(volume), st, types.SideBuy, sm.idgen)
}

// makeBoundaryOrder creates an accurrate order for the given one-tick interval which will exist at the edges
// of the adjusted expansion region.
func (sm *shapeMaker) getPos(st *num.Uint) int64 {
	// lets do the starting boundary order
	cu := sm.pool.lower
	if st.GTE(sm.pool.lower.high) {
		cu = sm.pool.upper
	}

	return cu.positionAtPrice(sm.pool.sqrt, st)
}

// calculateBoundaryOrders returns two orders which represent the edges of the adjust expansion region.
func (sm *shapeMaker) calculateBoundaryOrders() (*types.Order, *types.Order) {
	// we need to make sure that the orders at the boundary are the region are always accurate and not approximated
	// by that we mean that if the adjusted expansion region is [p1, p2] then we *always* have an order with price p1
	// and always have an order with price p2.
	//
	// The reason for this is that if we are in an auction and have a crossed-region of [p1, p2] and we don't ensure
	// we have orders *at* p1 and p2 then we create an inconsistency between the orderbook asking an AMM for its best bid/ask
	// and the orders it produces it that region.
	//
	// The two situations where we can miss boundary orders are:
	// - the expansion region is too large and we have to limit calculations and approximate orders
	// - the expansion region isn't divisible by `oneTick` and so we have to merge a sub-tick step in with the previous

	st := sm.from.Clone()
	nd := sm.to.Clone()

	sm.from.Add(st, sm.oneTick)
	sm.to.Sub(nd, sm.oneTick)

	if sm.from.GTE(sm.fairPrice) {
		sm.side = types.SideSell
	}

	bnd1 := sm.makeBoundaryOrder(st, sm.from)

	if sm.log.IsDebug() {
		sm.log.Debug("created boundary order",
			logging.String("price", bnd1.Price.String()),
			logging.String("side", bnd1.Side.String()),
			logging.Uint64("volume", bnd1.Size),
			logging.String("pool-party", sm.pool.AMMParty),
		)
	}

	bnd2 := sm.makeBoundaryOrder(sm.to, nd)

	if sm.log.IsDebug() {
		sm.log.Debug("created boundary order",
			logging.String("price", bnd2.Price.String()),
			logging.String("side", bnd2.Side.String()),
			logging.Uint64("volume", bnd2.Size),
			logging.String("pool-party", sm.pool.AMMParty),
		)
	}

	return bnd1, bnd2
}

// calculateStepSize looks at the size of the expansion region and increases the step size if it is too large.
func (sm *shapeMaker) calculateStepSize() {
	delta, _ := num.UintZero().Delta(sm.from, sm.to)
	delta.Div(delta, sm.oneTick)
	sm.step = sm.oneTick.Clone()

	fmt.Println("WWW caculate step size", delta, sm.pool.maxCalculationLevels)

	// if taking steps of one-tick doesn't breach the max-calculation levels then we can happily expand accurately
	if true || delta.LTE(sm.pool.maxCalculationLevels) {
		return
	}

	// if the expansion region is too wide, we need to approximate with bigger steps
	sm.step.Div(delta, sm.pool.maxCalculationLevels)
	sm.step.AddSum(num.UintOne()) // if delta / maxcals = 1.9 we're going to want steps of 2
	sm.step.Mul(sm.step, sm.oneTick)
	sm.approx = true
	fmt.Println("WWW caculate step size approx", sm.step)
	if sm.log.IsDebug() {
		sm.log.Debug("approximating orderbook expansion",
			logging.String("step", sm.step.String()),
			logging.String("pool-party", sm.pool.AMMParty),
		)
	}
}

// priceForStep returns a tradable order price for the volume between two price levels.
func (sm *shapeMaker) priceForStep(price1, price2 *num.Uint, pos1, pos2 int64, volume uint64) *num.Uint {
	if sm.side == types.SideBuy {
		if !sm.approx {
			return price1
		}
		return sm.pool.priceForVolumeAtPosition(volume, types.OtherSide(sm.side), pos2, price2)
	}

	if !sm.approx {
		return price2
	}

	return sm.pool.priceForVolumeAtPosition(volume, types.OtherSide(sm.side), pos1, price1)
}

// expandCurve walks along the given AMM curve between from -> to creating orders at each step.
func (sm *shapeMaker) expandCurve(cu *curve, from, to *num.Uint) {
	if sm.log.IsDebug() {
		sm.log.Debug("expanding pool curve",
			logging.Bool("lower-curve", cu.isLower),
			logging.String("low", cu.low.String()),
			logging.String("high", cu.high.String()),
			logging.String("from", sm.from.String()),
			logging.String("to", sm.to.String()),
		)
	}

	if cu.empty {
		return
	}

	from = num.Max(from, cu.low)
	to = num.Min(to, cu.high)

	// the price we have currently stepped to and the position of the AMM at that price
	current := from
	position := cu.positionAtPrice(sm.pool.sqrt, current)

	fairPrice := sm.fairPrice

	for current.LT(to) && current.LT(cu.high) {
		// take the next step
		next := num.UintZero().AddSum(current, sm.step)

		if sm.log.IsDebug() {
			sm.log.Debug("step taken",
				logging.String("current", current.String()),
				logging.String("next", next.String()),
			)
		}

		if num.UintZero().AddSum(next, sm.oneTick).GT(to) {
			// we step from current -> next, but if next is less that one-tick from the end
			// we will merge this into one bigger step so that we don't have a less-than one price level step
			next = to.Clone()
			if sm.log.IsDebug() {
				sm.log.Debug("increasing step size to prevent sub-tick price-level",
					logging.String("current", current.String()),
					logging.String("next-snapped", next.String()),
				)
			}
		}

		if sm.side == types.SideBuy && next.GT(fairPrice) && current.NEQ(fairPrice) {
			if sm.log.IsDebug() {
				sm.log.Debug("stepping over fair-price, splitting step",
					logging.String("fair-price", fairPrice.String()),
				)
			}

			if volume := uint64(num.DeltaV(position, sm.pos)); volume != 0 {
				price := sm.priceForStep(current, fairPrice, position, sm.pos, volume)
				sm.addOrder(volume, price, sm.side)
			}

			// we've step through fair-price now so orders will becomes sells
			sm.side = types.SideSell
			current = fairPrice
			position = sm.pos
		}

		nextPosition := cu.positionAtPrice(sm.pool.sqrt, num.Min(next, cu.high))
		volume := uint64(num.DeltaV(position, nextPosition))
		if volume != 0 {
			price := sm.priceForStep(current, next, position, nextPosition, volume)
			sm.addOrder(volume, price, sm.side)
		}

		// if we're calculating buys and we hit fair price, switch to sells
		if sm.side == types.SideBuy && next.GTE(fairPrice) {
			sm.side = types.SideSell
		}

		current = next
		position = nextPosition
	}
}

// adjustRegion takes the input to/from and increases or decreases the interval depending on the pool's bounds.
func (sm *shapeMaker) adjustRegion() bool {
	lower := sm.pool.lower.low
	upper := sm.pool.upper.high

	if sm.pool.closing() {
		// AMM is in reduce only mode so will only have orders between its fair-price and its base so shrink from/to to that region
		if sm.pos == 0 {
			// pool is closed and we're waiting for the next MTM to close, so it has no orders
			return false
		}

		if sm.pos > 0 {
			// only orders between fair-price -> base
			lower = sm.fairPrice.Clone()
			upper = sm.pool.lower.high.Clone()

			// if the AMM is super close to closing its position the delta between fair-price -> base
			// could be very small, but the upshot is we know it will only be one order and can calculate
			// directly
			if num.UintZero().Sub(upper, lower).LTE(sm.oneTick) {
				price := num.UintZero().Sub(sm.pool.lower.high, sm.oneTick)
				sm.addOrder(uint64(sm.pos), price, types.SideSell)
				return false
			}
		} else {
			// only orders between base -> fair-price
			upper = sm.fairPrice.Clone()
			lower = sm.pool.lower.high.Clone()

			if num.UintZero().Sub(upper, lower).LTE(sm.oneTick) {
				price := num.UintZero().Add(sm.pool.lower.high, sm.oneTick)
				sm.addOrder(uint64(-sm.pos), price, types.SideBuy)
				return false
			}
		}
	}

	if sm.from.GT(upper) || sm.to.LT(lower) {
		// expansion range is completely outside the pools ranges
		return false
	}

	// cap the range to the pool's bounds, there will be no orders outside of this
	from := num.Max(sm.from, lower)
	to := num.Min(sm.to, upper)

	// expansion is a point region *at* fair-price, there are no orders
	if from.EQ(to) && from.EQ(sm.fairPrice) {
		return false
	}

	switch {
	case sm.from.GT(sm.fairPrice):
		// if we are expanding entirely in the sell range to calculate the order at price `from`
		// we need to ask the AMM for volume in the range `from - 1 -> from` so we simply
		// sub one here to cover than.
		sm.side = types.SideSell
		from.Sub(from, sm.oneTick)
	case to.LT(sm.fairPrice):
		// if we are expanding entirely in the buy range to calculate the order at price `to`
		// we need to ask the AMM for volume in the range `to -> to + 1` so we simply
		// add one here to cover than.
		to.Add(to, sm.oneTick)
	case from.EQ(sm.fairPrice):
		// if we are starting the expansion at the fair-price all orders will be sells
		sm.side = types.SideSell
	}

	// we have the new range we will be expanding over, great
	sm.from = from
	sm.to = to
	return true
}

func (sm *shapeMaker) makeShape() ([]*types.Order, []*types.Order) {

	if !sm.adjustRegion() {
		// if there is no overlap between the input region and the AMM's bounds then there are no orders
		return sm.buys, sm.sells
	}

	// create accurate orders at the boundary of the adjusted region (even if we are going to make approximate internal steps)
	bnd1, bnd2 := sm.calculateBoundaryOrders()

	// we can add the start one now because it'll go at the beginning of the slice
	sm.appendOrder(bnd1)

	// work out the step size and if we'll be in approximate mode
	sm.calculateStepSize()

	// now walk across the lower curve
	sm.expandCurve(sm.pool.lower, sm.from, sm.to)

	// and walk across the upper curve
	sm.expandCurve(sm.pool.upper, sm.from, sm.to)

	// add the final boundary order we calculated earlier
	if bnd1.Price.NEQ(bnd2.Price) {
		sm.appendOrder(bnd2)
	}

	// add up all the volume
	total := uint64(0)
	for _, o := range sm.buys {
		total += o.Size
	}

	for _, o := range sm.sells {
		total += o.Size
	}

	if sm.log.IsDebug() {
		sm.log.Debug("pool expanded into orders",
			logging.Int("buys", len(sm.buys)),
			logging.Int("sells", len(sm.sells)),
		)
	}
	return sm.buys, sm.sells
}

func (p *Pool) OrderbookShape(from, to *num.Uint, idgen *idgeneration.IDGenerator) ([]*types.Order, []*types.Order) {
	return newShapeMaker(
		p.log,
		p,
		from,
		to,
		idgen).
		makeShape()
}
