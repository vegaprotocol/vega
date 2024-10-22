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
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderbookShape(t *testing.T) {
	t.Run("test orderbook shape pending AMM", testOrderbookShapePendingAMM)
	t.Run("test orderbook shape when AMM is 0", testOrderbookShapeZeroPosition)
	t.Run("test orderbook shape when AMM is long", testOrderbookShapeLong)
	t.Run("test orderbook shape when AMM is short", testOrderbookShapeShort)
	t.Run("test orderbook shape when calculations are capped", testOrderbookShapeLimited)
	t.Run("test orderbook shape step over fair price", testOrderbookShapeStepOverFairPrice)
	t.Run("test orderbook shape step fair price at boundary", testOrderbookShapeNoStepOverFairPrice)
	t.Run("test orderbook shape AMM reduce only", testOrderbookShapeReduceOnly)
	t.Run("test orderbook shape boundary order when approx", testOrderbookShapeBoundaryOrder)
	t.Run("test orderbook shape region not divisible by tick", testOrderbookSubTick)
	t.Run("test orderbook shape closing pool close to base", testClosingCloseToBase)
	t.Run("test orderbook shape point expansion at fair price", testPointExpansionAtFairPrice)
}

func testOrderbookShapePendingAMM(t *testing.T) {
	p := newTestPoolWithRanges(t, num.NewUint(7), num.NewUint(10), num.NewUint(13))
	defer p.ctrl.Finish()

	p.pool.status = types.AMMPoolStatusPending
	low := p.submission.Parameters.LowerBound
	high := p.submission.Parameters.UpperBound

	r := p.pool.OrderbookShape(low, high, nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))
}

func testOrderbookShapeZeroPosition(t *testing.T) {
	p := newTestPoolWithRanges(t, num.NewUint(7), num.NewUint(10), num.NewUint(13))
	defer p.ctrl.Finish()

	low := p.submission.Parameters.LowerBound
	base := p.submission.Parameters.Base
	high := p.submission.Parameters.UpperBound

	// when range [7, 10] expect orders at prices (7, 8, 9)
	// there will be no order at price 10 since that is the pools fair-price and it quotes +/-1 eitherside
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r := p.pool.OrderbookShape(low, base, nil)

	assertOrderPrices(t, r.Buys, types.SideBuy, 7, 9)
	assert.Equal(t, 0, len(r.Sells))

	// when range [7, 9] expect orders at prices (7, 8, 9)
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(low, num.NewUint(9), nil)
	assertOrderPrices(t, r.Buys, types.SideBuy, 7, 9)
	assert.Equal(t, 0, len(r.Sells))

	// when range [10, 13] expect orders at prices (11, 12, 13)
	// there will be no order at price 10 since that is the pools fair-price and it quotes +/-1 eitherside
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(base, high, nil)
	assert.Equal(t, 0, len(r.Buys))
	assertOrderPrices(t, r.Sells, types.SideSell, 11, 13)

	// when range [11, 13] expect orders at prices (11, 12, 13)
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(num.NewUint(11), high, nil)
	assert.Equal(t, 0, len(r.Buys))
	assertOrderPrices(t, r.Sells, types.SideSell, 11, 13)

	// whole range from [7, 10] will have buys (7, 8, 9) and sells (11, 12, 13)
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(low, high, nil)
	assertOrderPrices(t, r.Buys, types.SideBuy, 7, 9)
	assertOrderPrices(t, r.Sells, types.SideSell, 11, 13)

	// mid both curves spanning buys and sells, range from [8, 12] will have buys (8, 9) and sells (11, 12)
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(num.NewUint(8), num.NewUint(12), nil)
	assertOrderPrices(t, r.Buys, types.SideBuy, 8, 9)
	assertOrderPrices(t, r.Sells, types.SideSell, 11, 12)

	// range (8, 8) should return a single buy order at price 8, which is a bit counter intuitive
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(num.NewUint(8), num.NewUint(8), nil)
	assertOrderPrices(t, r.Buys, types.SideBuy, 8, 8)
	assert.Equal(t, 0, len(r.Sells))

	// range (10, 10) should return only the orders at the fair-price, which is 0 orders
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(num.NewUint(10), num.NewUint(10), nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))
}

func testOrderbookShapeLong(t *testing.T) {
	p := newTestPoolWithRanges(t, num.NewUint(7), num.NewUint(10), num.NewUint(13))
	defer p.ctrl.Finish()

	low := p.submission.Parameters.LowerBound
	base := p.submission.Parameters.Base
	high := p.submission.Parameters.UpperBound

	// AMM is long and will have a fair-price of 8
	position := int64(17980)
	ensurePosition(t, p.pos, position, num.UintZero())
	require.Equal(t, "8", p.pool.FairPrice().String())

	// range [7, 10] with have buy order (7) and sell orders (9, 10)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r := p.pool.OrderbookShape(low, base, nil)
	assertOrderPrices(t, r.Buys, types.SideBuy, 7, 7)
	assertOrderPrices(t, r.Sells, types.SideSell, 9, 10)

	// range [10, 13] with have sell orders (10, 11, 12, 13)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(base, high, nil)
	assert.Equal(t, 0, len(r.Buys))
	assertOrderPrices(t, r.Sells, types.SideSell, 10, 13)

	// whole range will have buys at (7) and sells at (9, 10, 11, 12, 13)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(low, high, nil)
	assertOrderPrices(t, r.Buys, types.SideBuy, 7, 7)
	assertOrderPrices(t, r.Sells, types.SideSell, 9, 13)

	// query at fair price returns no orders
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(num.NewUint(8), num.NewUint(8), nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))
}

func testOrderbookShapeShort(t *testing.T) {
	p := newTestPoolWithRanges(t, num.NewUint(7), num.NewUint(10), num.NewUint(13))
	defer p.ctrl.Finish()

	low := p.submission.Parameters.LowerBound
	base := p.submission.Parameters.Base
	high := p.submission.Parameters.UpperBound

	// AMM is short and will have a fair-price of 12
	position := int64(-20000)
	ensurePosition(t, p.pos, position, num.UintZero())
	require.Equal(t, "12", p.pool.FairPrice().String())

	// range [7, 10] with have buy order (7,8,9,10)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r := p.pool.OrderbookShape(low, base, nil)
	assertOrderPrices(t, r.Buys, types.SideBuy, 7, 10)
	assert.Equal(t, 0, len(r.Sells))

	// range [10, 13] with have buy orders (10, 11) and sell orders (13)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(base, high, nil)
	assertOrderPrices(t, r.Buys, types.SideBuy, 10, 11)
	assertOrderPrices(t, r.Sells, types.SideSell, 13, 13)

	// whole range will have buys at (7,8,9,10,11) and sells at (13)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(low, high, nil)
	assertOrderPrices(t, r.Buys, types.SideBuy, 7, 11)
	assertOrderPrices(t, r.Sells, types.SideSell, 13, 13)

	// query at fair price returns no orders
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(num.NewUint(12), num.NewUint(12), nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))
}

func testOrderbookShapeLimited(t *testing.T) {
	p := newTestPoolWithRanges(t, num.NewUint(20), num.NewUint(40), num.NewUint(60))
	defer p.ctrl.Finish()

	low := p.submission.Parameters.LowerBound
	base := p.submission.Parameters.Base
	high := p.submission.Parameters.UpperBound

	// position is zero but we're capping max calculations at ~10
	position := int64(0)
	p.pool.maxCalculationLevels = num.NewUint(10)

	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r := p.pool.OrderbookShape(low, base, nil)
	assert.Equal(t, 11, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))

	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(base, high, nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 11, len(r.Sells))

	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(low, high, nil)
	assert.Equal(t, 6, len(r.Buys))
	assert.Equal(t, 6, len(r.Sells))
}

func testOrderbookShapeStepOverFairPrice(t *testing.T) {
	p := newTestPoolWithRanges(t, num.NewUint(20), num.NewUint(40), num.NewUint(60))
	defer p.ctrl.Finish()

	low := p.submission.Parameters.LowerBound
	base := p.submission.Parameters.Base
	high := p.submission.Parameters.UpperBound

	// make levels of 10 makes the step price 2, and this position gives the pool a fair price of 25
	// when we take the step from 24 -> 26 we want to make sure we split that order into two, so we
	// will actually do maxCalculationLevels + 1 calculations but I think thats fine and keeps the calculations
	// simple
	position := int64(6000)
	p.pool.maxCalculationLevels = num.NewUint(10)
	ensurePosition(t, p.pos, position, num.UintZero())
	require.Equal(t, "26", p.pool.FairPrice().String())

	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r := p.pool.OrderbookShape(low, base, nil)
	assert.Equal(t, 4, len(r.Buys))
	assert.Equal(t, 8, len(r.Sells))

	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(base, high, nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 12, len(r.Sells))

	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(low, high, nil)
	assert.Equal(t, 3, len(r.Buys))
	assert.Equal(t, 10, len(r.Sells))
}

func testOrderbookShapeNoStepOverFairPrice(t *testing.T) {
	p := newTestPoolWithRanges(t, num.NewUint(20), num.NewUint(40), num.NewUint(60))
	defer p.ctrl.Finish()

	low := p.submission.Parameters.LowerBound
	base := p.submission.Parameters.Base
	high := p.submission.Parameters.UpperBound

	position := int64(0)
	p.pool.maxCalculationLevels = num.NewUint(6)

	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r := p.pool.OrderbookShape(low, base, nil)
	assert.Equal(t, 7, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))

	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(base, high, nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 7, len(r.Sells))

	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(low, high, nil)
	assert.Equal(t, 4, len(r.Buys))
	assert.Equal(t, 4, len(r.Sells))
}

func testOrderbookShapeReduceOnly(t *testing.T) {
	p := newTestPoolWithRanges(t, num.NewUint(7), num.NewUint(10), num.NewUint(13))
	defer p.ctrl.Finish()

	low := p.submission.Parameters.LowerBound
	base := p.submission.Parameters.Base
	high := p.submission.Parameters.UpperBound

	// pool is reduce only so will not have any orders above/below fair price depending on position
	p.pool.status = types.AMMPoolStatusReduceOnly

	// AMM is position 0 it will have no orders
	position := int64(0)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r := p.pool.OrderbookShape(low, base, nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))

	// AMM is long and will have a fair-price of 8 and so will only have orders from 8 -> base
	position = int64(17980)
	ensurePosition(t, p.pos, position, num.UintZero())
	require.Equal(t, "8", p.pool.FairPrice().String())

	// range [7, 13] will have only sellf orders (9, 10)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(low, high, nil)
	assert.Equal(t, 0, len(r.Buys))
	assertOrderPrices(t, r.Sells, types.SideSell, 9, 10)

	// AMM is short and will have a fair-price of 12
	position = int64(-20000)
	ensurePosition(t, p.pos, position, num.UintZero())
	require.Equal(t, "12", p.pool.FairPrice().String())

	// range [10, 13] with have buy orders (10, 11)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	r = p.pool.OrderbookShape(base, high, nil)
	assertOrderPrices(t, r.Buys, types.SideBuy, 10, 11)
	assert.Equal(t, 0, len(r.Sells))
}

func testOrderbookShapeBoundaryOrder(t *testing.T) {
	p := newTestPoolWithRanges(t, num.NewUint(100), num.NewUint(200), num.NewUint(300))
	defer p.ctrl.Finish()

	midlow := num.NewUint(150)
	midhigh := num.NewUint(250)

	position := int64(0)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)

	// limit the number of orders in the expansion
	p.pool.maxCalculationLevels = num.NewUint(5)

	r := p.pool.OrderbookShape(midlow, midhigh, nil)
	assert.Equal(t, 4, len(r.Buys))
	assert.Equal(t, 4, len(r.Sells))

	// we're in approximate mode but we still require an exact order at the boundaries of the shape range
	// check that the price for the first by is midlow, and the last sell is midhigh
	assert.Equal(t, midlow.String(), r.Buys[0].Price.String())
	assert.Equal(t, midhigh.String(), r.Sells[(len(r.Sells)-1)].Price.String())
}

func testOrderbookSubTick(t *testing.T) {
	p := newTestPoolWithSubmission(t, num.DecimalFromFloat(1), num.DecimalFromFloat(100),
		&types.SubmitAMM{
			CommitmentAmount: num.NewUint(10000000),
			Parameters: &types.ConcentratedLiquidityParameters{
				LowerBound: num.NewUint(10),
				Base:       num.NewUint(15),
				UpperBound: num.NewUint(20),
			},
		},
		0,
	)

	defer p.ctrl.Finish()

	// limit the number of orders in the expansion
	p.pool.maxCalculationLevels = num.NewUint(1000)

	position := int64(1000)
	ensurePositionN(t, p.pos, position, num.UintZero(), 4)

	// fair-price should be 1483, and so best buy should be 1383 (fair-price minus one-tick)
	bp, _ := p.pool.BestPrice(types.SideBuy)
	require.Equal(t, "1383", bp.String())

	// now pretend we are in auction and we have a sell order at 1000, so we need to expand the crossed
	// region of 1000 -> 1383
	from := num.NewUint(1000)
	to := num.NewUint(1383)
	r := p.pool.OrderbookShape(from, to, nil)

	assert.Equal(t, 4, len(r.Buys))
	assert.Equal(t, bp.String(), r.Buys[3].Price.String())

	assert.Equal(t, 0, len(r.Sells))
}

func testClosingCloseToBase(t *testing.T) {
	p := newTestPoolWithSubmission(t, num.DecimalFromFloat(1), num.DecimalFromFloat(100),
		&types.SubmitAMM{
			CommitmentAmount: num.NewUint(10000000),
			Parameters: &types.ConcentratedLiquidityParameters{
				LowerBound: num.NewUint(10),
				Base:       num.NewUint(15),
				UpperBound: num.NewUint(20),
			},
		},
		0,
	)

	defer p.ctrl.Finish()

	// its reducing
	p.pool.status = types.AMMPoolStatusReduceOnly

	// and it is long one
	position := int64(1)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)

	// now pretend we are in auction and we have a sell order at 1000, so we need to expand the crossed
	// region of 1000 -> 1383
	from := num.NewUint(1000)
	to := num.NewUint(2000)
	r := p.pool.OrderbookShape(from, to, nil)

	// should have one sell of volume 1
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 1, len(r.Sells))
	assert.Equal(t, 1, int(r.Sells[0].Size))
	assert.Equal(t, "14", r.Sells[0].OriginalPrice.String())

	// and it is short one
	position = int64(-1)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)

	r = p.pool.OrderbookShape(from, to, nil)

	// should have one sell of volume 1
	assert.Equal(t, 1, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))
	assert.Equal(t, 1, int(r.Buys[0].Size))
	assert.Equal(t, "16", r.Buys[0].OriginalPrice.String())

	// no position
	position = int64(0)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)

	r = p.pool.OrderbookShape(from, to, nil)

	// should have one sell of volume 1
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))
}

func testPointExpansionAtFairPrice(t *testing.T) {
	p := newTestPoolWithRanges(t, num.NewUint(7), num.NewUint(10), num.NewUint(13))
	defer p.ctrl.Finish()

	base := p.submission.Parameters.Base

	// range [10, 10] fair price is 10, no orders
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r := p.pool.OrderbookShape(base, base, nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))

	// now try with a one sided curve where the input range shrinks to a point-expansion
	p = newTestPoolWithRanges(t, num.NewUint(7), num.NewUint(10), nil)
	defer p.ctrl.Finish()

	// range [10, 1000] but sell curve is empty so effective range is [10, 10] at fair-price
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(base, num.NewUint(1000), nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))

	// now try with a one sided curve where the input range shrinks to a point-expansion
	p = newTestPoolWithRanges(t, nil, num.NewUint(10), num.NewUint(13))
	defer p.ctrl.Finish()

	// range [1, 10] but buy curve is empty so effective range is [10, 10] at fair-price
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(num.NewUint(1), base, nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))
}

func TestOrderbookShapeSparseAMM(t *testing.T) {
	p := newTestPoolWithOpts(t, num.DecimalOne(), num.NewUint(99), num.NewUint(198), num.NewUint(297), num.NewUint(1000), 100)

	defer p.ctrl.Finish()

	low := p.submission.Parameters.LowerBound
	base := p.submission.Parameters.Base
	high := p.submission.Parameters.UpperBound

	// when range [99, 198] expect orders at ticks of 9 from 99 -> 189
	// there will be no order at price 200 since that is the pools fair-price and it quotes +/-1 eitherside
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r := p.pool.OrderbookShape(low, base, nil)
	assert.Equal(t, 14, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))

	// check boundary orders
	assert.Equal(t, "99", r.Buys[0].Price.String())
	assert.Equal(t, "191", r.Buys[(len(r.Buys)-1)].Price.String())

	// when range [99, 191] expect orders at prices 100 -> 191
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(low, num.NewUint(189), nil)
	assert.Equal(t, 14, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))

	// check boundary orders
	assert.Equal(t, "99", r.Buys[0].Price.String())
	assert.Equal(t, "189", r.Buys[(len(r.Buys)-1)].Price.String())

	// when range [198, 297] expect orders at prices 207 -> 297
	// there will be no order at price 198 since that is the pools fair-price and it quotes +/-1 eitherside
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(base, high, nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 12, len(r.Sells))

	// check boundary orders
	assert.Equal(t, "203", r.Sells[0].Price.String())
	assert.Equal(t, "297", r.Sells[(len(r.Sells)-1)].Price.String())

	// when range [207, 297] expect orders at prices 207 -> 297
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(num.NewUint(207), high, nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 11, len(r.Sells))

	// check boundary orders
	assert.Equal(t, "207", r.Sells[0].Price.String())
	assert.Equal(t, "297", r.Sells[(len(r.Sells)-1)].Price.String())

	// range (8, 8) should return a single buy order at price 8, which is a bit counter intuitive
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(num.NewUint(117), num.NewUint(117), nil)
	assertOrderPrices(t, r.Buys, types.SideBuy, 117, 117)
	assert.Equal(t, 0, len(r.Sells))

	// range (10, 10) should return only the orders at the fair-price, which is 0 orders
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	r = p.pool.OrderbookShape(num.NewUint(198), num.NewUint(198), nil)
	assert.Equal(t, 0, len(r.Buys))
	assert.Equal(t, 0, len(r.Sells))
}

func TestOrderbookShapeSparseAMMBoundaryOrders(t *testing.T) {
	submit := &types.SubmitAMM{
		AMMBaseCommand: types.AMMBaseCommand{
			Party:             vgcrypto.RandomHash(),
			MarketID:          vgcrypto.RandomHash(),
			SlippageTolerance: num.DecimalFromFloat(0.1),
		},
		CommitmentAmount: num.MustUintFromString("5960289358452040000000", 10),
		Parameters: &types.ConcentratedLiquidityParameters{
			Base:                 num.NewUint(195836),
			LowerBound:           num.NewUint(92079),
			UpperBound:           num.NewUint(246109),
			LeverageAtLowerBound: ptr.From(num.DecimalFromFloat(95.13295934001242)),
			LeverageAtUpperBound: ptr.From(num.DecimalFromFloat(86.1204552842962)),
		},
	}
	p := newTestPoolWithSubmission(t, num.DecimalFromInt64(1000), num.DecimalFromFloat(10000000000000000), submit, 100)
	defer p.ctrl.Finish()

	ensurePositionN(t, p.pos, 0, nil, -1)
	r := p.pool.OrderbookShape(
		num.MustUintFromString("1958381716019393098944", 10),
		num.MustUintFromString("4897350000000000000000", 10),
		nil,
	)
	assert.Equal(t, 0, len(r.Buys))
	require.Equal(t, "1958381716019393098944", r.Sells[0].Price.String())
	require.Equal(t, "2461090000000000000000", r.Sells[len(r.Sells)-1].Price.String())
}
