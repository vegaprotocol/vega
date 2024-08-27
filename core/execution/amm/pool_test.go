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
	"code.vegaprotocol.io/vega/logging"

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
	t.Run("test volume between prices when closing", testTradeableVolumeInRangeClosing)
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
			expectedVolume: 1335,
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
			expectedVolume: 985,
			position:       350,
		},
		{
			name:           "sell trade causes sign to flip and partial volume across both curves",
			price1:         num.NewUint(500),
			price2:         num.NewUint(3500),
			side:           types.SideSell,
			expectedVolume: 1052,
			position:       -350,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensurePositionN(t, p.pos, tt.position, num.UintZero(), 1)
			volume := p.pool.TradableVolumeInRange(tt.side, tt.price1, tt.price2)
			assert.Equal(t, int(tt.expectedVolume), int(volume))
		})
	}
}

func testTradeableVolumeInRangeClosing(t *testing.T) {
	p := newTestPool(t)
	defer p.ctrl.Finish()

	// pool is reducing its
	p.pool.status = types.AMMPoolStatusReduceOnly

	tests := []struct {
		name           string
		price1         *num.Uint
		price2         *num.Uint
		position       int64
		side           types.Side
		expectedVolume uint64
		nposcalls      int
	}{
		{
			name:           "0 position, 0 buy volume",
			price1:         num.NewUint(1800),
			price2:         num.NewUint(2200),
			side:           types.SideBuy,
			expectedVolume: 0,
			nposcalls:      1,
		},
		{
			name:           "0 position, 0 sell volume",
			price1:         num.NewUint(1800),
			price2:         num.NewUint(2200),
			side:           types.SideSell,
			expectedVolume: 0,
			nposcalls:      1,
		},
		{
			name:           "long position, 0 volume for incoming SELL",
			price1:         num.NewUint(1800),
			price2:         num.NewUint(2200),
			side:           types.SideSell,
			position:       10,
			expectedVolume: 0,
			nposcalls:      1,
		},
		{
			name:           "long position, 10 volume for incoming BUY",
			price1:         num.NewUint(1800),
			price2:         num.NewUint(2200),
			side:           types.SideBuy,
			position:       10,
			expectedVolume: 10,
			nposcalls:      2,
		},
		{
			name:           "short position, 0 volume for incoming BUY",
			price1:         num.NewUint(1800),
			price2:         num.NewUint(2200),
			side:           types.SideBuy,
			position:       -10,
			expectedVolume: 0,
			nposcalls:      1,
		},
		{
			name:           "short position, 10 volume for incoming SELL",
			price1:         num.NewUint(1800),
			price2:         num.NewUint(2200),
			side:           types.SideSell,
			position:       -10,
			expectedVolume: 10,
			nposcalls:      2,
		},
		{
			name:           "asking for SELL volume but for prices outside of price ranges",
			price1:         num.NewUint(2000),
			price2:         num.NewUint(2200),
			side:           types.SideBuy,
			position:       10,
			expectedVolume: 0,
			nposcalls:      2,
		},
		{
			name:           "asking for BUY volume but for prices outside of price ranges",
			price1:         num.NewUint(1800),
			price2:         num.NewUint(1850),
			side:           types.SideSell,
			position:       -10,
			expectedVolume: 0,
			nposcalls:      2,
		},
		{
			name:           "asking for partial closing volume when long",
			price1:         num.NewUint(1800),
			price2:         num.NewUint(1850),
			side:           types.SideBuy,
			position:       702,
			expectedVolume: 186,
			nposcalls:      2,
		},
		{
			name:           "asking for partial closing volume when short",
			price1:         num.NewUint(2100),
			price2:         num.NewUint(2150),
			side:           types.SideSell,
			position:       -635,
			expectedVolume: 155,
			nposcalls:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensurePositionN(t, p.pos, tt.position, num.UintZero(), tt.nposcalls)
			volume := p.pool.TradableVolumeInRange(tt.side, tt.price1, tt.price2)
			assert.Equal(t, int(tt.expectedVolume), int(volume))
		})
	}
}

func TestTradeableVolumeWhenAtBoundary(t *testing.T) {
	// from ticket 11389 this replicates a scenario found during fuzz testing
	submit := &types.SubmitAMM{
		AMMBaseCommand: types.AMMBaseCommand{
			Party:             vgcrypto.RandomHash(),
			MarketID:          vgcrypto.RandomHash(),
			SlippageTolerance: num.DecimalFromFloat(0.1),
		},
		CommitmentAmount: num.MustUintFromString("2478383748073213000000", 10),
		Parameters: &types.ConcentratedLiquidityParameters{
			Base:                 num.NewUint(676540),
			LowerBound:           num.NewUint(671272),
			UpperBound:           nil,
			LeverageAtLowerBound: ptr.From(num.DecimalFromFloat(39.1988064541227)),
			LeverageAtUpperBound: nil,
		},
	}

	p := newTestPoolWithSubmission(t,
		num.DecimalFromInt64(1000),
		num.DecimalFromFloat(10000000000000000),
		submit,
	)
	defer p.ctrl.Finish()

	// when position is zero fair-price should be the base
	ensurePositionN(t, p.pos, 0, num.UintZero(), 2)
	fp := p.pool.BestPrice(nil)
	assert.Equal(t, "6765400000000000000000", fp.String())

	fullLong := 12546

	// volume from base -> low is 12546, but in reality it is 12546.4537027400278, but we can only trade int volume.
	volume := p.pool.TradableVolumeInRange(types.SideSell, num.MustUintFromString("6712720000000000000000", 10), num.MustUintFromString("6765400000000000000000", 10))
	assert.Equal(t, fullLong, int(volume))

	// now lets pretend the AMM has fully traded out in that direction, best price will be near but not quite the lower bound
	ensurePositionN(t, p.pos, int64(fullLong), num.UintZero(), 2)
	fp = p.pool.BestPrice(nil)
	assert.Equal(t, "6712721893865935337785", fp.String())
	assert.True(t, fp.GTE(num.MustUintFromString("6712720000000000000000", 10)))

	// now the fair-price is not *quite* on the lower boundary but the volume between it at the lower bound should be 0.
	volume = p.pool.TradableVolumeInRange(types.SideSell, num.MustUintFromString("6712720000000000000000", 10), fp)
	assert.Equal(t, 0, int(volume))
}

func testPoolPositionFactor(t *testing.T) {
	p := newTestPoolWithPositionFactor(t, num.DecimalFromInt64(1000))
	defer p.ctrl.Finish()

	ensurePositionN(t, p.pos, 0, num.UintZero(), 1)
	volume := p.pool.TradableVolumeInRange(types.SideBuy, num.NewUint(2000), num.NewUint(2200))
	// with position factot of 1 the volume is 635
	assert.Equal(t, int(635395), int(volume))

	ensurePositionN(t, p.pos, 0, num.UintZero(), 1)
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
	ensurePositionN(t, p.pos, 0, num.UintZero(), 1)
	volume := p.pool.TradableVolumeInRange(types.SideBuy, num.NewUint(2000), num.NewUint(2200))
	assert.Equal(t, int(635), int(volume))

	// empty side returns no volume
	ensurePositionN(t, p.pos, 0, num.UintZero(), 1)
	volume = p.pool.TradableVolumeInRange(types.SideSell, num.NewUint(1800), num.NewUint(2000))
	assert.Equal(t, int(0), int(volume))

	// pool with short position and incoming sell only reports volume up to base
	// empty side returns no volume
	ensurePositionN(t, p.pos, -10, num.UintZero(), 1)
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
		logging.NewTestLogger(),
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

	ensurePositionN(t, p.pos, pos, num.UintZero(), 1)
	volume := p.pool.TradableVolumeInRange(types.SideSell, base, low)
	assert.Equal(t, int(702), int(volume))

	ensurePositionN(t, p.pos, pos, num.UintZero(), 1)
	volume = p.pool.TradableVolumeInRange(types.SideBuy, up, base)
	assert.Equal(t, int(635), int(volume))

	lowmid := num.NewUint(1900)
	upmid := num.NewUint(2100)

	ensurePositionN(t, p.pos, pos, num.UintZero(), 1)
	volume = p.pool.TradableVolumeInRange(types.SideSell, low, lowmid)
	assert.Equal(t, int(365), int(volume))

	ensurePositionN(t, p.pos, pos, num.UintZero(), 1)
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
		logging.NewTestLogger(),
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

func newTestPoolWithSubmission(t *testing.T, positionFactor, priceFactor num.Decimal, submit *types.SubmitAMM) *tstPool {
	t.Helper()
	ctrl := gomock.NewController(t)
	col := mocks.NewMockCollateral(ctrl)
	pos := mocks.NewMockPosition(ctrl)

	sqrter := &Sqrter{cache: map[string]num.Decimal{}}

	pool, err := NewPool(
		logging.NewTestLogger(),
		vgcrypto.RandomHash(),
		vgcrypto.RandomHash(),
		vgcrypto.RandomHash(),
		submit,
		sqrter.sqrt,
		col,
		pos,
		&types.RiskFactor{
			Short: num.DecimalFromFloat(0.009937604878885509),
			Long:  num.DecimalFromFloat(0.00984363574304481),
		},
		&types.ScalingFactors{
			InitialMargin: num.DecimalFromFloat(1.5), // this is 1/0.8 which is margin_usage_at_bound_above in the note-book
		},
		num.DecimalFromFloat(0.001),
		priceFactor,
		positionFactor,
		num.NewUint(1000),
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
