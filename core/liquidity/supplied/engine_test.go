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

package supplied_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/liquidity/supplied"
	"code.vegaprotocol.io/vega/core/liquidity/supplied/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var (
	MarkPrice                          = num.NewUint(103)
	MarkPriceD                         = MarkPrice.ToDecimal()
	DefaultInRangeProbabilityOfTrading = num.DecimalFromFloat(.5)
	Horizon                            = num.DecimalFromFloat(0.001)
	TickSize                           = num.NewUint(1)
)

func TestLiquidityScore(t *testing.T) {
	minLpPrice, maxLpPrice := num.UintOne(), num.MaxUint()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)

	minPMPrice := num.NewWrappedDecimal(num.NewUint(89), num.DecimalFromInt64(89))
	maxPMPrice := num.NewWrappedDecimal(num.NewUint(111), num.DecimalFromInt64(111))

	// No orders
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPMPrice, maxPMPrice).AnyTimes()
	statevarEngine := stubs.NewStateVar()
	engine := supplied.NewEngine(riskModel, priceMonitor, "asset1", "market1", statevarEngine, logging.NewTestLogger(), num.DecimalFromInt64(1))
	require.NotNil(t, engine)

	f := func() (num.Decimal, num.Decimal, error) { return MarkPriceD, MarkPriceD, nil }
	engine.SetGetStaticPricesFunc(f)

	liquidity := engine.CalculateLiquidityScore([]*types.Order{}, MarkPriceD, MarkPriceD, minLpPrice, maxLpPrice)
	require.True(t, liquidity.IsZero())

	// 1 buy, no sells
	buyOrder1 := &types.Order{
		Price:     num.NewUint(102),
		Size:      30,
		Remaining: 25,
		Side:      types.SideBuy,
	}

	buyOrder1Prob := num.DecimalFromFloat(0.256)
	sellOrder1Prob := num.DecimalFromFloat(0.33)
	sellOrder2Prob := num.DecimalFromFloat(0.17)

	sellOrder1 := &types.Order{
		Price:     num.NewUint(105),
		Size:      15,
		Remaining: 11,
		Side:      types.SideSell,
	}
	sellOrder2 := &types.Order{
		Price:     num.NewUint(104),
		Size:      60,
		Remaining: 60,
		Side:      types.SideSell,
	}

	riskModel.EXPECT().ProbabilityOfTrading(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(best, order, min num.Decimal, max num.Decimal, yFrac num.Decimal, isBid bool, applyMinMax bool) num.Decimal {
		if best.Equal(MarkPriceD) && order.Sub(buyOrder1.Price.ToDecimal()).Abs().LessThanOrEqual(num.DecimalFromFloat(0.1)) && isBid {
			return buyOrder1Prob
		}
		if best.Equal(MarkPriceD) && order.Sub(sellOrder1.Price.ToDecimal()).Abs().LessThanOrEqual(num.DecimalFromFloat(0.1)) && !isBid {
			return sellOrder1Prob
		}
		if best.Equal(MarkPriceD) && order.Sub(sellOrder2.Price.ToDecimal()).Abs().LessThanOrEqual(num.DecimalFromFloat(0.1)) && !isBid {
			return sellOrder2Prob
		}
		if order.LessThanOrEqual(num.DecimalZero()) {
			return num.DecimalZero()
		}
		if order.GreaterThanOrEqual(num.DecimalFromInt64(2).Mul(best)) {
			return num.DecimalZero()
		}
		return num.DecimalFromFloat(0.5)
	})

	statevarEngine.NewEvent("asset1", "market1", statevar.EventTypeAuctionEnded)
	liquidity2 := engine.CalculateLiquidityScore([]*types.Order{}, MarkPriceD, MarkPriceD, minLpPrice, maxLpPrice)
	require.True(t, liquidity2.IsZero())

	buyOrder1Size := num.DecimalFromInt64(int64(buyOrder1.Remaining))
	buyLiquidityScore := buyOrder1Prob.Mul(DefaultInRangeProbabilityOfTrading).Mul(buyOrder1Size)
	buySideTotalSize := num.DecimalZero().Add(buyOrder1Size)

	buyLiquidityScore = buyLiquidityScore.Div(buySideTotalSize)

	sellOrder1Size := num.DecimalFromInt64(int64(sellOrder1.Remaining))
	sellLiquidityScore := sellOrder1Prob.Mul(DefaultInRangeProbabilityOfTrading).Mul(sellOrder1Size)
	sellSideTotalSize := num.DecimalZero().Add(sellOrder1Size)

	sellOrder2Size := num.DecimalFromInt64(int64(sellOrder2.Remaining))
	sellLiquidityScore = sellLiquidityScore.Add(sellOrder2Prob.Mul(DefaultInRangeProbabilityOfTrading).Mul(sellOrder2Size))
	sellSideTotalSize = sellSideTotalSize.Add(sellOrder2Size)

	sellLiquidityScore = sellLiquidityScore.Div(sellSideTotalSize)

	expectedScore := min(buyLiquidityScore, sellLiquidityScore)
	liquidity3 := engine.CalculateLiquidityScore([]*types.Order{buyOrder1, sellOrder1, sellOrder2}, MarkPriceD, MarkPriceD, minLpPrice, maxLpPrice)
	require.True(t, expectedScore.Equal(liquidity3))

	// 2 buys, 2 sells
	buyOrder2 := &types.Order{
		Price:     num.NewUint(102),
		Size:      600,
		Remaining: 599,
		Side:      types.SideBuy,
	}
	buyOrder2Prob := num.DecimalFromFloat(0.256)

	//	buyLiquidity += buyOrder2.Price.Float64() * float64(buyOrder2.Remaining) * buyOrder2Prob
	buyOrder2Size := num.DecimalFromInt64(int64(buyOrder2.Remaining))
	buyLiquidityScore = buyOrder1Prob.Mul(DefaultInRangeProbabilityOfTrading).Mul(buyOrder1Size).Add(buyOrder2Prob.Mul(DefaultInRangeProbabilityOfTrading).Mul(buyOrder2Size))
	buySideTotalSize = num.DecimalZero().Add(buyOrder1Size).Add(buyOrder2Size)

	buyLiquidityScore = buyLiquidityScore.Div(buySideTotalSize)

	expectedScore = min(buyLiquidityScore, sellLiquidityScore)
	liquidity4 := engine.CalculateLiquidityScore([]*types.Order{buyOrder1, sellOrder1, sellOrder2, buyOrder2}, MarkPriceD, MarkPriceD, minLpPrice, maxLpPrice)
	require.True(t, expectedScore.Equal(liquidity4))

	// Orders outside PM range (but within LP range)

	// add orders outwith the PM bounds
	buyOrder3 := &types.Order{
		Price:     num.UintZero().Sub(minPMPrice.Representation(), num.UintOne()),
		Size:      123,
		Remaining: 45,
		Side:      types.SideBuy,
	}
	sellOrder3 := &types.Order{
		Price:     num.UintZero().Add(maxPMPrice.Representation(), num.UintOne()),
		Size:      345,
		Remaining: 67,
		Side:      types.SideSell,
	}

	// liquidity should drop as the volume-weighted PoT of trading within the LP range drops (some orders included in the score now have PoT==0)
	liquidity5 := engine.CalculateLiquidityScore([]*types.Order{buyOrder1, sellOrder1, sellOrder2, buyOrder2, sellOrder3, buyOrder3}, MarkPriceD, MarkPriceD, minLpPrice, maxLpPrice)
	require.True(t, liquidity5.LessThan(liquidity4))

	// Orders outside LP range (but within PM range)

	// set bounds at prices of orders furtherst away form the mid
	minLpPrice = buyOrder2.Price
	maxLpPrice = sellOrder1.Price

	// add orders outwith the LP bounds
	buyOrder3 = &types.Order{
		Price:     num.UintZero().Sub(minLpPrice, num.UintOne()),
		Size:      123,
		Remaining: 45,
		Side:      types.SideBuy,
	}
	sellOrder3 = &types.Order{
		Price:     num.UintZero().Add(maxLpPrice, num.UintOne()),
		Size:      345,
		Remaining: 67,
		Side:      types.SideSell,
	}

	// liquidity shouldn't change
	liquidity6 := engine.CalculateLiquidityScore([]*types.Order{buyOrder1, sellOrder1, sellOrder2, buyOrder2, sellOrder3, buyOrder3}, MarkPriceD, MarkPriceD, minLpPrice, maxLpPrice)
	require.Equal(t, liquidity4, liquidity6)
}

func min(d1, d2 num.Decimal) num.Decimal {
	if d1.LessThan(d2) {
		return d1
	}
	return d2
}
