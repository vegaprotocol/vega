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
	"strconv"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/amm/mocks"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAMMPool(t *testing.T) {
	t.Run("test volume between prices", testTradeableVolumeInRange)
	t.Run("test best price", testBestPrice)
	t.Run("test pool logic with position factor", testPoolPositionFactor)
	t.Run("test one sided pool", testOneSidedPool)
	t.Run("test near zero volume curve triggers and error", testNearZeroCurveErrors)
}

func TestOrderbookShape(t *testing.T) {
	t.Run("test orderbook shape when AMM is 0", testOrderbookShapeZeroPosition)
	t.Run("test orderbook shape when AMM is long", testOrderbookShapeLong)
	t.Run("test orderbook shape when AMM is short", testOrderbookShapeShort)
	t.Run("test orderbook shape when calculations are capped", testOrderbookShapeLimited)
	t.Run("test orderbook shape step over fair price", testOrderbookShapeStepOverFairPrice)
	t.Run("test orderbook shape step fair price at boundary", testOrderbookShapeNoStepOverFairPrice)
	t.Run("test orderbook shape AMM reduce only", testOrderbookShapeReduceOnly)
}

func testTradeableVolumeInRange(t *testing.T) {
	p := newTestPool(t)
	defer p.ctrl.Finish()

	tests := []struct {
		name           string
		price1         *num.Uint
		price2         *num.Uint
		position       int64
		side           types.Side
		expectedVolume uint64
	}{
		{
			name:           "full volume upper curve",
			price1:         num.NewUint(2000),
			price2:         num.NewUint(2200),
			side:           types.SideBuy,
			expectedVolume: 635,
		},
		{
			name:           "full volume upper curve with bound creep",
			price1:         num.NewUint(1500),
			price2:         num.NewUint(3500),
			side:           types.SideBuy,
			expectedVolume: 635,
		},
		{
			name:           "full volume lower curve",
			price1:         num.NewUint(1800),
			price2:         num.NewUint(2000),
			side:           types.SideSell,
			expectedVolume: 702,
		},
		{
			name:           "full volume lower curve with bound creep",
			price1:         num.NewUint(500),
			price2:         num.NewUint(2500),
			side:           types.SideSell,
			expectedVolume: 702,
		},
		{
			name:           "buy trade causes sign to flip and full volume crosses curves",
			price1:         num.NewUint(500),
			price2:         num.NewUint(3500),
			side:           types.SideBuy,
			expectedVolume: 1337,
			position:       700, // position at full lower boundary, incoming is by so whole volume of both curves is available
		},
		{
			name:           "sell trade causes sign to flip and full volume crosses curves",
			price1:         num.NewUint(500),
			price2:         num.NewUint(3500),
			side:           types.SideSell,
			expectedVolume: 1337,
			position:       -700, // position at full upper boundary, incoming is by so whole volume of both curves is available
		},
		{
			name:           "buy trade causes sign to flip and partial volume across both curves",
			price1:         num.NewUint(500),
			price2:         num.NewUint(3500),
			side:           types.SideBuy,
			expectedVolume: 986,
			position:       350,
		},
		{
			name:           "sell trade causes sign to flip and partial volume across both curves",
			price1:         num.NewUint(500),
			price2:         num.NewUint(3500),
			side:           types.SideSell,
			expectedVolume: 1053,
			position:       -350,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensurePositionN(t, p.pos, tt.position, num.UintZero(), 2)
			volume := p.pool.TradableVolumeInRange(tt.side, tt.price1, tt.price2)
			assert.Equal(t, int(tt.expectedVolume), int(volume))
		})
	}
}

func testPoolPositionFactor(t *testing.T) {
	p := newTestPoolWithPositionFactor(t, num.DecimalFromInt64(1000))
	defer p.ctrl.Finish()

	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	volume := p.pool.TradableVolumeInRange(types.SideBuy, num.NewUint(2000), num.NewUint(2200))
	// with position factot of 1 the volume is 635
	assert.Equal(t, int(635395), int(volume))

	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	volume = p.pool.TradableVolumeInRange(types.SideSell, num.NewUint(1800), num.NewUint(2000))
	// with position factot of 1 the volume is 702
	assert.Equal(t, int(702411), int(volume))

	ensurePositionN(t, p.pos, -1, num.NewUint(2000), 1)
	// now best price should be the same as if the factor were 1, since its a price and not a volume
	fairPrice := p.pool.BestPrice(nil)
	assert.Equal(t, "2001", fairPrice.String())
}

func testBestPrice(t *testing.T) {
	p := newTestPool(t)
	defer p.ctrl.Finish()

	tests := []struct {
		name          string
		position      int64
		balance       uint64
		expectedPrice string
		order         *types.Order
	}{
		{
			name:          "best price is base price when position is zero",
			expectedPrice: "2000",
		},
		{
			name:          "best price positive position",
			expectedPrice: "1999",
			position:      1,
			balance:       100000,
		},

		{
			name:          "fair price negative position",
			expectedPrice: "2001",
			position:      -1,
			balance:       100000,
		},
		{
			name:          "best price incoming buy",
			expectedPrice: "2000",
			position:      1,
			balance:       100000,
			order: &types.Order{
				Side: types.SideBuy,
				Size: 1,
			},
		},
		{
			name:          "best price incoming buy",
			expectedPrice: "1998",
			position:      1,
			balance:       100000,
			order: &types.Order{
				Side: types.SideSell,
				Size: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := tt.order
			ensurePosition(t, p.pos, tt.position, num.UintZero())
			fairPrice := p.pool.BestPrice(order)
			assert.Equal(t, tt.expectedPrice, fairPrice.String())
		})
	}
}

func testOneSidedPool(t *testing.T) {
	// a pool with no liquidity below
	p := newTestPoolWithRanges(t, nil, num.NewUint(2000), num.NewUint(2200))
	defer p.ctrl.Finish()

	// side with liquidity returns volume
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	volume := p.pool.TradableVolumeInRange(types.SideBuy, num.NewUint(2000), num.NewUint(2200))
	assert.Equal(t, int(635), int(volume))

	// empty side returns no volume
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	volume = p.pool.TradableVolumeInRange(types.SideSell, num.NewUint(1800), num.NewUint(2000))
	assert.Equal(t, int(0), int(volume))

	// pool with short position and incoming sell only reports volume up to base
	// empty side returns no volume
	ensurePositionN(t, p.pos, -10, num.UintZero(), 2)
	volume = p.pool.TradableVolumeInRange(types.SideSell, num.NewUint(1800), num.NewUint(2200))
	assert.Equal(t, int(10), int(volume))

	// fair price at 0 position is still ok
	ensurePosition(t, p.pos, 0, num.UintZero())
	price := p.pool.BestPrice(nil)
	assert.Equal(t, "2000", price.String())

	// fair price at short position is still ok
	ensurePosition(t, p.pos, -10, num.UintZero())
	price = p.pool.BestPrice(nil)
	assert.Equal(t, "2003", price.String())

	// fair price when long should panic since AMM should never be able to get into that state
	// fair price at short position is still ok
	ensurePosition(t, p.pos, 10, num.UintZero())
	assert.Panics(t, func() { p.pool.BestPrice(nil) })
}

func testNearZeroCurveErrors(t *testing.T) {
	baseCmd := types.AMMBaseCommand{
		Party:             vgcrypto.RandomHash(),
		MarketID:          vgcrypto.RandomHash(),
		SlippageTolerance: num.DecimalFromFloat(0.1),
	}

	submit := &types.SubmitAMM{
		AMMBaseCommand:   baseCmd,
		CommitmentAmount: num.NewUint(1000),
		Parameters: &types.ConcentratedLiquidityParameters{
			Base:                 num.NewUint(1900),
			LowerBound:           num.NewUint(1800),
			UpperBound:           num.NewUint(2000),
			LeverageAtLowerBound: ptr.From(num.DecimalFromFloat(50)),
			LeverageAtUpperBound: ptr.From(num.DecimalFromFloat(50)),
		},
	}
	// test that creating a pool with a near zero volume curve will error
	pool, err := newBasicPoolWithSubmit(t, submit)
	assert.Nil(t, pool)
	assert.ErrorContains(t, err, "insufficient commitment - less than one volume at price levels on lower curve")

	// test that a pool with higher commitment amount will not error
	submit.CommitmentAmount = num.NewUint(100000)
	pool, err = newBasicPoolWithSubmit(t, submit)
	assert.NotNil(t, pool)
	assert.NoError(t, err)

	// test that amending a pool to a near zero volume curve will error
	amend := &types.AmendAMM{
		AMMBaseCommand:   baseCmd,
		CommitmentAmount: num.NewUint(100),
	}

	_, err = pool.Update(
		amend,
		&types.RiskFactor{Short: num.DecimalFromFloat(0.02), Long: num.DecimalFromFloat(0.02)},
		&types.ScalingFactors{InitialMargin: num.DecimalFromFloat(1.25)},
		num.DecimalZero(),
	)
	assert.ErrorContains(t, err, "insufficient commitment - less than one volume at price levels on lower curve")

	amend.CommitmentAmount = num.NewUint(1000000)
	_, err = pool.Update(
		amend,
		&types.RiskFactor{Short: num.DecimalFromFloat(0.02), Long: num.DecimalFromFloat(0.02)},
		&types.ScalingFactors{InitialMargin: num.DecimalFromFloat(1.25)},
		num.DecimalZero(),
	)
	assert.NoError(t, err)
}

func testOrderbookShapeZeroPosition(t *testing.T) {
	p := newTestPoolWithRanges(t, num.NewUint(7), num.NewUint(10), num.NewUint(13))
	defer p.ctrl.Finish()

	low := p.submission.Parameters.LowerBound
	base := p.submission.Parameters.Base
	high := p.submission.Parameters.UpperBound

	// when range [7, 10] expect orders at prices (7, 8, 9)
	// there will be no order at price 10 since that is the pools fair-price and it quotes +/-1 eitherside
	ensurePosition(t, p.pos, 0, num.UintZero())
	buys, sells := p.pool.OrderbookShape(low, base, nil)
	assertOrderPrices(t, buys, types.SideBuy, 7, 9)
	assert.Equal(t, 0, len(sells))

	// when range [7, 9] expect orders at prices (7, 8, 9)
	ensurePosition(t, p.pos, 0, num.UintZero())
	buys, sells = p.pool.OrderbookShape(low, num.NewUint(9), nil)
	assertOrderPrices(t, buys, types.SideBuy, 7, 9)
	assert.Equal(t, 0, len(sells))

	// when range [10, 13] expect orders at prices (11, 12, 13)
	// there will be no order at price 10 since that is the pools fair-price and it quotes +/-1 eitherside
	ensurePosition(t, p.pos, 0, num.UintZero())
	buys, sells = p.pool.OrderbookShape(base, high, nil)
	assert.Equal(t, 0, len(buys))
	assertOrderPrices(t, sells, types.SideSell, 11, 13)

	// when range [11, 13] expect orders at prices (11, 12, 13)
	ensurePosition(t, p.pos, 0, num.UintZero())
	buys, sells = p.pool.OrderbookShape(num.NewUint(11), high, nil)
	assert.Equal(t, 0, len(buys))
	assertOrderPrices(t, sells, types.SideSell, 11, 13)

	// whole range from [7, 10] will have buys (7, 8, 9) and sells (11, 12, 13)
	ensurePosition(t, p.pos, 0, num.UintZero())
	buys, sells = p.pool.OrderbookShape(low, high, nil)
	assertOrderPrices(t, buys, types.SideBuy, 7, 9)
	assertOrderPrices(t, sells, types.SideSell, 11, 13)

	// mid both curves spanning buys and sells, range from [8, 12] will have buys (8, 9) and sells (11, 12)
	ensurePosition(t, p.pos, 0, num.UintZero())
	buys, sells = p.pool.OrderbookShape(num.NewUint(8), num.NewUint(12), nil)
	assertOrderPrices(t, buys, types.SideBuy, 8, 9)
	assertOrderPrices(t, sells, types.SideSell, 11, 12)

	// range (8, 8) should return a single buy order at price 8, which is a bit counter intuitive
	ensurePosition(t, p.pos, 0, num.UintZero())
	buys, sells = p.pool.OrderbookShape(num.NewUint(8), num.NewUint(8), nil)
	assertOrderPrices(t, buys, types.SideBuy, 8, 8)
	assert.Equal(t, 0, len(sells))

	// range (10, 10) should return only the orders at the fair-price, which is 0 orders
	ensurePosition(t, p.pos, 0, num.UintZero())
	buys, sells = p.pool.OrderbookShape(num.NewUint(10), num.NewUint(10), nil)
	assert.Equal(t, 0, len(buys))
	assert.Equal(t, 0, len(sells))
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
	require.Equal(t, "8", p.pool.BestPrice(nil).String())

	// range [7, 10] with have buy order (7) and sell orders (9, 10)
	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells := p.pool.OrderbookShape(low, base, nil)
	assertOrderPrices(t, buys, types.SideBuy, 7, 7)
	assertOrderPrices(t, sells, types.SideSell, 9, 10)

	// range [10, 13] with have sell orders (10, 11, 12, 13)
	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells = p.pool.OrderbookShape(base, high, nil)
	assert.Equal(t, 0, len(buys))
	assertOrderPrices(t, sells, types.SideSell, 10, 13)

	// whole range will have buys at (7) and sells at (9, 10, 11, 12, 13)
	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells = p.pool.OrderbookShape(low, high, nil)
	assertOrderPrices(t, buys, types.SideBuy, 7, 7)
	assertOrderPrices(t, sells, types.SideSell, 9, 13)

	// query at fair price returns no orders
	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells = p.pool.OrderbookShape(num.NewUint(8), num.NewUint(8), nil)
	assert.Equal(t, 0, len(buys))
	assert.Equal(t, 0, len(sells))
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
	require.Equal(t, "12", p.pool.BestPrice(nil).String())

	// range [7, 10] with have buy order (7,8,9,10)
	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells := p.pool.OrderbookShape(low, base, nil)
	assertOrderPrices(t, buys, types.SideBuy, 7, 10)
	assert.Equal(t, 0, len(sells))

	// range [10, 13] with have buy orders (10, 11) and sell orders (13)
	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells = p.pool.OrderbookShape(base, high, nil)
	assertOrderPrices(t, buys, types.SideBuy, 10, 11)
	assertOrderPrices(t, sells, types.SideSell, 13, 13)

	// whole range will have buys at (7,8,9,10,11) and sells at (13)
	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells = p.pool.OrderbookShape(low, high, nil)
	assertOrderPrices(t, buys, types.SideBuy, 7, 11)
	assertOrderPrices(t, sells, types.SideSell, 13, 13)

	// query at fair price returns no orders
	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells = p.pool.OrderbookShape(num.NewUint(12), num.NewUint(12), nil)
	assert.Equal(t, 0, len(buys))
	assert.Equal(t, 0, len(sells))
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

	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells := p.pool.OrderbookShape(low, base, nil)
	assert.Equal(t, 10, len(buys))
	assert.Equal(t, 0, len(sells))

	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells = p.pool.OrderbookShape(base, high, nil)
	assert.Equal(t, 0, len(buys))
	assert.Equal(t, 10, len(sells))

	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells = p.pool.OrderbookShape(low, high, nil)
	assert.Equal(t, 5, len(buys))
	assert.Equal(t, 5, len(sells))
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
	position := int64(7000)
	p.pool.maxCalculationLevels = num.NewUint(10)
	ensurePosition(t, p.pos, position, num.UintZero())
	require.Equal(t, "25", p.pool.BestPrice(nil).String())

	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	buys, sells := p.pool.OrderbookShape(low, base, nil)
	assert.Equal(t, 3, len(buys))
	assert.Equal(t, 8, len(sells))

	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells = p.pool.OrderbookShape(base, high, nil)
	assert.Equal(t, 0, len(buys))
	assert.Equal(t, 11, len(sells))

	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	buys, sells = p.pool.OrderbookShape(low, high, nil)
	assert.Equal(t, 2, len(buys))
	assert.Equal(t, 9, len(sells))
}

func testOrderbookShapeNoStepOverFairPrice(t *testing.T) {
	p := newTestPoolWithRanges(t, num.NewUint(20), num.NewUint(40), num.NewUint(60))
	defer p.ctrl.Finish()

	low := p.submission.Parameters.LowerBound
	base := p.submission.Parameters.Base
	high := p.submission.Parameters.UpperBound

	position := int64(0)
	p.pool.maxCalculationLevels = num.NewUint(6)

	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells := p.pool.OrderbookShape(low, base, nil)
	assert.Equal(t, 7, len(buys))
	assert.Equal(t, 0, len(sells))

	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells = p.pool.OrderbookShape(base, high, nil)
	assert.Equal(t, 0, len(buys))
	assert.Equal(t, 7, len(sells))

	ensurePosition(t, p.pos, position, num.UintZero())
	buys, sells = p.pool.OrderbookShape(low, high, nil)
	assert.Equal(t, 4, len(buys))
	assert.Equal(t, 4, len(sells))
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
	buys, sells := p.pool.OrderbookShape(low, base, nil)
	assert.Equal(t, 0, len(buys))
	assert.Equal(t, 0, len(sells))

	// AMM is long and will have a fair-price of 8 and so will only have orders from 8 -> base
	position = int64(17980)
	ensurePosition(t, p.pos, position, num.UintZero())
	require.Equal(t, "8", p.pool.BestPrice(nil).String())

	// range [7, 13] will have only sellf orders (9, 10)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	buys, sells = p.pool.OrderbookShape(low, high, nil)
	assert.Equal(t, 0, len(buys))
	assertOrderPrices(t, sells, types.SideSell, 9, 10)

	// AMM is short and will have a fair-price of 12
	position = int64(-20000)
	ensurePosition(t, p.pos, position, num.UintZero())
	require.Equal(t, "12", p.pool.BestPrice(nil).String())

	// range [10, 13] with have buy orders (10, 11)
	ensurePositionN(t, p.pos, position, num.UintZero(), 2)
	buys, sells = p.pool.OrderbookShape(base, high, nil)
	assertOrderPrices(t, buys, types.SideBuy, 10, 11)
	assert.Equal(t, 0, len(sells))
}

func assertOrderPrices(t *testing.T, orders []*types.Order, side types.Side, st, nd int) {
	t.Helper()
	require.Equal(t, nd-st+1, len(orders))
	for i, o := range orders {
		price := st + i
		assert.Equal(t, side, o.Side)
		assert.Equal(t, strconv.FormatInt(int64(price), 10), o.Price.String())
	}
}

func newBasicPoolWithSubmit(t *testing.T, submit *types.SubmitAMM) (*Pool, error) {
	t.Helper()
	ctrl := gomock.NewController(t)
	col := mocks.NewMockCollateral(ctrl)
	pos := mocks.NewMockPosition(ctrl)

	sqrter := &Sqrter{cache: map[string]num.Decimal{}}

	return NewPool(
		vgcrypto.RandomHash(),
		vgcrypto.RandomHash(),
		vgcrypto.RandomHash(),
		submit,
		sqrter.sqrt,
		col,
		pos,
		&types.RiskFactor{
			Short: num.DecimalFromFloat(0.02),
			Long:  num.DecimalFromFloat(0.02),
		},
		&types.ScalingFactors{
			InitialMargin: num.DecimalFromFloat(1.25), // this is 1/0.8 which is margin_usage_at_bound_above in the note-book
		},
		num.DecimalZero(),
		num.DecimalOne(),
		num.DecimalOne(),
		num.NewUint(10000),
	)
}

func ensurePositionN(t *testing.T, p *mocks.MockPosition, pos int64, averageEntry *num.Uint, times int) {
	t.Helper()

	if times < 0 {
		p.EXPECT().GetPositionsByParty(gomock.Any()).AnyTimes().Return(
			[]events.MarketPosition{&marketPosition{size: pos, averageEntry: averageEntry}},
		)
	} else {
		p.EXPECT().GetPositionsByParty(gomock.Any()).Times(times).Return(
			[]events.MarketPosition{&marketPosition{size: pos, averageEntry: averageEntry}},
		)
	}
}

func ensurePosition(t *testing.T, p *mocks.MockPosition, pos int64, averageEntry *num.Uint) {
	t.Helper()

	ensurePositionN(t, p, pos, averageEntry, 1)
}

func ensureBalancesN(t *testing.T, col *mocks.MockCollateral, balance uint64, times int) {
	t.Helper()

	// split the balance equally across general and margin
	split := balance / 2
	gen := &types.Account{
		Balance: num.NewUint(split),
	}
	mar := &types.Account{
		Balance: num.NewUint(balance - split),
	}

	if times < 0 {
		col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(gen, nil)
		col.EXPECT().GetPartyMarginAccount(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(mar, nil)
	} else {
		col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(times).Return(gen, nil)
		col.EXPECT().GetPartyMarginAccount(gomock.Any(), gomock.Any(), gomock.Any()).Times(times).Return(mar, nil)
	}
}

func ensureBalances(t *testing.T, col *mocks.MockCollateral, balance uint64) {
	t.Helper()
	ensureBalancesN(t, col, balance, 1)
}

func TestNotebook(t *testing.T) {
	// Note that these were verified using Tom's jupyter notebook, so don't go arbitrarily changing the numbers
	// without re-verifying!

	p := newTestPool(t)
	defer p.ctrl.Finish()

	base := num.NewUint(2000)
	low := num.NewUint(1800)
	up := num.NewUint(2200)

	pos := int64(0)

	ensurePositionN(t, p.pos, pos, num.UintZero(), 2)
	volume := p.pool.TradableVolumeInRange(types.SideSell, base, low)
	assert.Equal(t, int(702), int(volume))

	ensurePositionN(t, p.pos, pos, num.UintZero(), 2)
	volume = p.pool.TradableVolumeInRange(types.SideBuy, up, base)
	assert.Equal(t, int(635), int(volume))

	lowmid := num.NewUint(1900)
	upmid := num.NewUint(2100)

	ensurePositionN(t, p.pos, pos, num.UintZero(), 2)
	volume = p.pool.TradableVolumeInRange(types.SideSell, low, lowmid)
	assert.Equal(t, int(365), int(volume))

	ensurePositionN(t, p.pos, pos, num.UintZero(), 2)
	volume = p.pool.TradableVolumeInRange(types.SideBuy, upmid, up)
	assert.Equal(t, int(306), int(volume))

	ensurePosition(t, p.pos, -500, upmid.Clone())
	fairPrice := p.pool.BestPrice(nil)
	assert.Equal(t, "2155", fairPrice.String())

	ensurePosition(t, p.pos, 500, lowmid.Clone())
	fairPrice = p.pool.BestPrice(nil)
	assert.Equal(t, "1854", fairPrice.String())

	// fair price is 2000 and the AMM quotes a best-buy at 1999 so incoming SELL should have a price <= 1999
	ensurePositionN(t, p.pos, 0, lowmid.Clone(), 2)
	price := p.pool.PriceForVolume(100, types.SideSell)
	assert.Equal(t, "1984", price.String())

	// fair price is 2000 and the AMM quotes a best-buy at 2001 so incoming BUY should have a price >= 2001
	ensurePositionN(t, p.pos, 0, lowmid.Clone(), 2)
	price = p.pool.PriceForVolume(100, types.SideBuy)
	assert.Equal(t, "2014", price.String())
}

type tstPool struct {
	pool       *Pool
	col        *mocks.MockCollateral
	pos        *mocks.MockPosition
	ctrl       *gomock.Controller
	submission *types.SubmitAMM
}

func newTestPool(t *testing.T) *tstPool {
	t.Helper()
	return newTestPoolWithPositionFactor(t, num.DecimalOne())
}

func newTestPoolWithRanges(t *testing.T, low, base, high *num.Uint) *tstPool {
	t.Helper()
	return newTestPoolWithOpts(t, num.DecimalOne(), low, base, high)
}

func newTestPoolWithPositionFactor(t *testing.T, positionFactor num.Decimal) *tstPool {
	t.Helper()
	return newTestPoolWithOpts(t, positionFactor, num.NewUint(1800), num.NewUint(2000), num.NewUint(2200))
}

func newTestPoolWithOpts(t *testing.T, positionFactor num.Decimal, low, base, high *num.Uint) *tstPool {
	t.Helper()
	ctrl := gomock.NewController(t)
	col := mocks.NewMockCollateral(ctrl)
	pos := mocks.NewMockPosition(ctrl)

	sqrter := &Sqrter{cache: map[string]num.Decimal{}}

	submit := &types.SubmitAMM{
		AMMBaseCommand: types.AMMBaseCommand{
			Party:             vgcrypto.RandomHash(),
			MarketID:          vgcrypto.RandomHash(),
			SlippageTolerance: num.DecimalFromFloat(0.1),
		},
		// 0000000000000
		CommitmentAmount: num.NewUint(100000),
		Parameters: &types.ConcentratedLiquidityParameters{
			Base:                 base,
			LowerBound:           low,
			UpperBound:           high,
			LeverageAtLowerBound: ptr.From(num.DecimalFromFloat(50)),
			LeverageAtUpperBound: ptr.From(num.DecimalFromFloat(50)),
		},
	}

	pool, err := NewPool(
		vgcrypto.RandomHash(),
		vgcrypto.RandomHash(),
		vgcrypto.RandomHash(),
		submit,
		sqrter.sqrt,
		col,
		pos,
		&types.RiskFactor{
			Short: num.DecimalFromFloat(0.02),
			Long:  num.DecimalFromFloat(0.02),
		},
		&types.ScalingFactors{
			InitialMargin: num.DecimalFromFloat(1.25), // this is 1/0.8 which is margin_usage_at_bound_above in the note-book
		},
		num.DecimalZero(),
		num.DecimalOne(),
		positionFactor,
		num.NewUint(100000),
	)
	assert.NoError(t, err)

	return &tstPool{
		submission: submit,
		pool:       pool,
		col:        col,
		pos:        pos,
		ctrl:       ctrl,
	}
}

type marketPosition struct {
	size         int64
	averageEntry *num.Uint
}

func (m marketPosition) AverageEntryPrice() *num.Uint { return m.averageEntry.Clone() }
func (m marketPosition) Party() string                { return "" }
func (m marketPosition) Size() int64                  { return m.size }
func (m marketPosition) Buy() int64                   { return 0 }
func (m marketPosition) Sell() int64                  { return 0 }
func (m marketPosition) Price() *num.Uint             { return num.UintZero() }
func (m marketPosition) BuySumProduct() *num.Uint     { return num.UintZero() }
func (m marketPosition) SellSumProduct() *num.Uint    { return num.UintZero() }
func (m marketPosition) VWBuy() *num.Uint             { return num.UintZero() }
func (m marketPosition) VWSell() *num.Uint            { return num.UintZero() }
