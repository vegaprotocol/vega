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

package common_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAMMStateSnapshot(t *testing.T) {
	testLiquidity := newMarketLiquidity(t)
	// set fee factor to 1, so fees are not paid out based on score.
	testLiquidity.marketLiquidity.SetELSFeeFraction(num.DecimalOne())
	testLiquidity.liquidityEngine.EXPECT().ReadyForFeesAllocation(gomock.Any()).Return(false).AnyTimes()
	testLiquidity.liquidityEngine.EXPECT().UpdateAverageLiquidityScores(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	testLiquidity.liquidityEngine.EXPECT().OnStakeToCcyVolumeUpdate(gomock.Any()).AnyTimes()
	testLiquidity.marketLiquidity.OnStakeToCcyVolumeUpdate(num.DecimalOne())

	testLiquidity.orderBook.EXPECT().GetBestStaticAskPrice().Return(num.NewUint(200), nil).AnyTimes()
	testLiquidity.orderBook.EXPECT().GetBestStaticBidPrice().Return(num.NewUint(100), nil).AnyTimes()
	testLiquidity.liquidityEngine.EXPECT().GetPartyLiquidityScore(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(num.DecimalOne()).AnyTimes()

	party1 := vgcrypto.RandomHash()
	party2 := vgcrypto.RandomHash()
	amms := map[string]common.AMMPool{
		party1: dummyAMM{},
	}

	// get some AMM's to create some state
	testLiquidity.amm.EXPECT().GetAMMPoolsBySubAccount().Return(amms)

	testLiquidity.marketLiquidity.OnTick(context.Background(), time.Now())

	state := testLiquidity.marketLiquidity.GetState()
	assert.Equal(t, int64(1), state.Tick)
	assert.Equal(t, int64(1), state.Amm[0].Tick)

	// next tick we have a new AMM enter the ring
	amms[party2] = dummyAMM{}
	testLiquidity.amm.EXPECT().GetAMMPoolsBySubAccount().Return(amms)

	testLiquidity.marketLiquidity.OnTick(context.Background(), time.Now())

	state = testLiquidity.marketLiquidity.GetState()
	assert.Equal(t, int64(2), state.Tick)
	if state.Amm[0].Party == party1 {
		assert.Equal(t, int64(1), state.Amm[1].Tick)
		assert.Equal(t, int64(2), state.Amm[0].Tick)
	} else {
		assert.Equal(t, int64(1), state.Amm[0].Tick)
		assert.Equal(t, int64(2), state.Amm[1].Tick)
	}
}

type dummyAMM struct{}

func (d dummyAMM) OrderbookShape(from, to *num.Uint, idgen *idgeneration.IDGenerator) *types.OrderbookShapeResult {
	return &types.OrderbookShapeResult{}
}

func (d dummyAMM) LiquidityFee() num.Decimal {
	return num.DecimalZero()
}

func (d dummyAMM) CommitmentAmount() *num.Uint {
	return num.UintOne()
}
