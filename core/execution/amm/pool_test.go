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

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/amm/mocks"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAMMPool(t *testing.T) {
	t.Run("test volume between prices", testVolumeBetweenPrices)
	t.Run("test trade price", testTradePrice)
}

func testVolumeBetweenPrices(t *testing.T) {
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
			side:           types.SideSell,
			expectedVolume: 3049821,
		},
		{
			name:           "full volume upper curve with bound creep",
			price1:         num.NewUint(1500),
			price2:         num.NewUint(3500),
			side:           types.SideSell,
			expectedVolume: 3049821,
		},
		{
			name:           "full volume lower curve",
			price1:         num.NewUint(1800),
			price2:         num.NewUint(2000),
			side:           types.SideBuy,
			expectedVolume: 3340281,
		},
		{
			name:           "full volume lower curve with bound creep",
			price1:         num.NewUint(500),
			price2:         num.NewUint(2500),
			side:           types.SideBuy,
			expectedVolume: 3340281,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensurePosition(t, p, tt.position, nil)
			volume := p.pool.VolumeBetweenPrices(tt.side, tt.price1, tt.price2)
			assert.Equal(t, int(tt.expectedVolume), int(volume))
		})
	}
}

func testTradePrice(t *testing.T) {
	p := newTestPool(t)
	defer p.ctrl.Finish()

	tests := []struct {
		name              string
		position          int64
		balance           uint64
		averageEntryPrice *num.Uint
		expectedPrice     string
		order             *types.Order
	}{
		{
			name:              "fair price is base price when position is zero",
			expectedPrice:     "2000",
			averageEntryPrice: num.UintZero(),
		},
		{
			name:              "fair price positive position",
			expectedPrice:     "881",
			position:          100,
			balance:           100000000000,
			averageEntryPrice: num.NewUint(2000),
		},
		{
			name:              "fair price negative position",
			expectedPrice:     "96",
			position:          -100,
			balance:           100000000000,
			averageEntryPrice: num.NewUint(2000),
		},
		{
			name:              "trade price incoming buy",
			expectedPrice:     "882",
			position:          100,
			balance:           100000000000,
			averageEntryPrice: num.NewUint(2000),
			order: &types.Order{
				Side: types.SideBuy,
				Size: 1,
			},
		},
		{
			name:              "trade price incoming buy",
			expectedPrice:     "880",
			position:          100,
			balance:           100000000000,
			averageEntryPrice: num.NewUint(2000),
			order: &types.Order{
				Side: types.SideSell,
				Size: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := tt.order
			if tt.order == nil {
				// size zero means we asking for a fair price
				order = &types.Order{Side: types.SideBuy, Size: 0}
			}

			ensurePosition(t, p, tt.position, tt.averageEntryPrice)

			if tt.position != 0 {
				ensureBalances(t, p, tt.balance)
			}
			fairPrice := p.pool.TradePrice(order)
			assert.Equal(t, tt.expectedPrice, fairPrice.String())
		})
	}
}

func ensurePosition(t *testing.T, p *tstPool, pos int64, averageEntry *num.Uint) {
	t.Helper()

	p.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(1).Return(
		[]events.MarketPosition{&marketPosition{size: pos, averageEntry: averageEntry}},
	)
}

func ensureBalances(t *testing.T, p *tstPool, balance uint64) {
	t.Helper()

	// split the balance equall across general and margin
	split := balance / 2
	gen := &types.Account{
		Balance: num.NewUint(split),
	}
	mar := &types.Account{
		Balance: num.NewUint(balance - split),
	}

	p.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(gen, nil)
	p.col.EXPECT().GetPartyMarginAccount(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(mar, nil)
}

type tstPool struct {
	pool *Pool
	col  *mocks.MockCollateral
	pos  *mocks.MockPosition
	ctrl *gomock.Controller
}

func newTestPool(t *testing.T) *tstPool {
	t.Helper()
	ctrl := gomock.NewController(t)
	col := mocks.NewMockCollateral(ctrl)
	pos := mocks.NewMockPosition(ctrl)

	sqrter := &Sqrter{cache: map[string]*num.Uint{}}

	submit := &types.SubmitAMM{
		AMMBaseCommand: types.AMMBaseCommand{
			Party:             vgcrypto.RandomHash(),
			MarketID:          vgcrypto.RandomHash(),
			SlippageTolerance: num.DecimalFromFloat(0.1),
		},
		CommitmentAmount: num.NewUint(10000000000),
		Parameters: &types.ConcentratedLiquidityParameters{
			Base:                    num.NewUint(2000),
			LowerBound:              num.NewUint(1800),
			UpperBound:              num.NewUint(2200),
			MarginRatioAtLowerBound: nil,
			MarginRatioAtUpperBound: nil,
		},
	}

	pool := NewPool(
		vgcrypto.RandomHash(),
		vgcrypto.RandomHash(),
		vgcrypto.RandomHash(),
		submit,
		sqrter.sqrt,
		col,
		pos,
		&types.RiskFactor{
			Short: num.DecimalFromFloat(0.08),
			Long:  num.DecimalFromFloat(0.08),
		},
		&types.ScalingFactors{
			InitialMargin: num.DecimalFromFloat(1.5),
		},
		num.DecimalOne(),
		num.UintOne(),
	)

	return &tstPool{
		pool: pool,
		col:  col,
		pos:  pos,
		ctrl: ctrl,
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
