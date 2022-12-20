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

	// add orders outwith the LP bounds
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

	// liquidity shouldn't change
	liquidity5 := engine.CalculateLiquidityScore([]*types.Order{buyOrder1, sellOrder1, sellOrder2, buyOrder2, sellOrder3, buyOrder3}, MarkPriceD, MarkPriceD, minLpPrice, maxLpPrice)
	require.Equal(t, liquidity4, liquidity5)

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

func Test_InteralConsistency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minLpPrice := num.NewUint(89)
	maxLpPrice := num.NewUint(111)

	limitOrders := []*types.Order{}

	buy := &supplied.LiquidityOrder{
		Price:   minLpPrice,
		Details: &types.LiquidityOrder{Proportion: 1},
	}
	buyShapes := []*supplied.LiquidityOrder{
		buy,
	}

	sell := &supplied.LiquidityOrder{
		Price:   maxLpPrice,
		Details: &types.LiquidityOrder{Proportion: 1},
	}

	sellShapes := []*supplied.LiquidityOrder{
		sell,
	}

	engine := supplied.NewEngine(riskModel, priceMonitor, "asset1", "market1", stubs.NewStateVar(), logging.NewTestLogger(), num.DecimalFromInt64(1))
	require.NotNil(t, engine)

	// Negative liquidity obligation -> 0 sizes on all orders
	liquidityObligation := num.NewUint(100)
	engine.CalculateLiquidityImpliedVolumes(liquidityObligation.Clone(), limitOrders, minLpPrice, maxLpPrice, buyShapes, sellShapes)

	var zero uint64
	require.Less(t, zero, buy.LiquidityImpliedVolume)
	require.Less(t, zero, sell.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity := num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(allOrders, minLpPrice, maxLpPrice))
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))
}

func TestCalculateLiquidityImpliedSizes_NoLimitOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minLpPrice := num.NewUint(89)
	maxLpPrice := num.NewUint(111)

	limitOrders := []*types.Order{}

	validBuy1 := &supplied.LiquidityOrder{
		Price:   minLpPrice,
		Details: &types.LiquidityOrder{Proportion: 20},
	}

	validBuy2 := &supplied.LiquidityOrder{
		Price:   num.Sum(minLpPrice, num.NewUint(1)),
		Details: &types.LiquidityOrder{Proportion: 30},
	}
	buyShapes := []*supplied.LiquidityOrder{
		validBuy1,
		validBuy2,
	}
	validSell1 := &supplied.LiquidityOrder{
		Price:   num.UintZero().Sub(maxLpPrice, num.NewUint(1)),
		Details: &types.LiquidityOrder{Proportion: 11},
	}
	validSell2 := &supplied.LiquidityOrder{
		Price:   maxLpPrice,
		Details: &types.LiquidityOrder{Proportion: 22},
	}
	sellShapes := []*supplied.LiquidityOrder{
		validSell1,
		validSell2,
	}

	engine := supplied.NewEngine(riskModel, priceMonitor, "asset1", "market1", stubs.NewStateVar(), logging.NewTestLogger(), num.DecimalFromInt64(1))
	require.NotNil(t, engine)

	// No liquidity obligation -> 0 sizes on all orders
	liquidityObligation := num.NewUint(0)
	engine.CalculateLiquidityImpliedVolumes(liquidityObligation, limitOrders, minLpPrice, maxLpPrice, buyShapes, sellShapes)

	var zero uint64
	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity := num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(allOrders, minLpPrice, maxLpPrice))
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))

	// 0 liquidity obligation -> 0 sizes on all orders
	liquidityObligation = num.NewUint(0)
	engine.CalculateLiquidityImpliedVolumes(liquidityObligation, limitOrders, minLpPrice, maxLpPrice, buyShapes, sellShapes)

	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(allOrders, minLpPrice, maxLpPrice))
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))

	// Positive liquidity obligation -> positive sizes on orders -> suplied liquidity >= liquidity obligation
	liquidityObligation = num.NewUint(250)
	engine.CalculateLiquidityImpliedVolumes(liquidityObligation, limitOrders, minLpPrice, maxLpPrice, buyShapes, sellShapes)

	loDec := liquidityObligation.ToDecimal()
	vb1Prop := num.DecimalFromFloat(float64(validBuy1.Details.Proportion))
	vb2Prop := num.DecimalFromFloat(float64(validBuy2.Details.Proportion))
	vs1Prop := num.DecimalFromFloat(float64(validSell1.Details.Proportion))
	vs2Prop := num.DecimalFromFloat(float64(validSell2.Details.Proportion))
	vb1Price := validBuy1.Price.ToDecimal()
	vb2Price := validBuy2.Price.ToDecimal()
	vs1Price := validSell1.Price.ToDecimal()
	vs2Price := validSell2.Price.ToDecimal()
	expVolVB1 := loDec.Mul(vb1Prop).Div(vb1Prop.Add(vb2Prop)).Div(vb1Price).Ceil()
	expVolVB2 := loDec.Mul(vb2Prop).Div(vb1Prop.Add(vb2Prop)).Div(vb2Price).Ceil()

	expVolVS1 := loDec.Mul(vs1Prop).Div(vs1Prop.Add(vs2Prop)).Div(vs1Price).Ceil()
	expVolVS2 := loDec.Mul(vs2Prop).Div(vs1Prop.Add(vs2Prop)).Div(vs2Price).Ceil()

	expectedVolumeValidBuy1, _ := num.UintFromDecimal(expVolVB1)
	expectedVolumeValidBuy2, _ := num.UintFromDecimal(expVolVB2)
	expectedVolumeValidSell1, _ := num.UintFromDecimal(expVolVS1)
	expectedVolumeValidSell2, _ := num.UintFromDecimal(expVolVS2)
	require.Equal(t, expectedVolumeValidBuy1.Uint64(), validBuy1.LiquidityImpliedVolume)
	require.Equal(t, expectedVolumeValidBuy2.Uint64(), validBuy2.LiquidityImpliedVolume)
	require.Equal(t, expectedVolumeValidSell1.Uint64(), validSell1.LiquidityImpliedVolume)
	require.Equal(t, expectedVolumeValidSell2.Uint64(), validSell2.LiquidityImpliedVolume)

	// Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(allOrders, minLpPrice, maxLpPrice))
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))
	require.True(t, totalSuppliedLiquidity.LT(liquidityObligation.Mul(liquidityObligation, num.NewUint(2))))
}

func TestCalculateLiquidityImpliedSizes_WithLimitOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minLpPrice := num.NewUint(89)
	maxLpPrice := num.NewUint(111)

	validBuy1 := &supplied.LiquidityOrder{
		Price:   minLpPrice,
		Details: &types.LiquidityOrder{Proportion: 20},
	}
	validBuy2 := &supplied.LiquidityOrder{
		Price:   num.Sum(minLpPrice, num.NewUint(1)),
		Details: &types.LiquidityOrder{Proportion: 20},
	}
	buyShapes := []*supplied.LiquidityOrder{
		validBuy1,
		validBuy2,
	}
	validSell1 := &supplied.LiquidityOrder{
		Price:   num.UintZero().Sub(maxLpPrice, num.NewUint(1)),
		Details: &types.LiquidityOrder{Proportion: 11},
	}
	validSell2 := &supplied.LiquidityOrder{
		Price:   maxLpPrice,
		Details: &types.LiquidityOrder{Proportion: 22},
	}
	sellShapes := []*supplied.LiquidityOrder{
		validSell1,
		validSell2,
	}

	engine := supplied.NewEngine(riskModel, priceMonitor, "asset1", "market1", stubs.NewStateVar(), logging.NewTestLogger(), num.DecimalFromInt64(1))
	require.NotNil(t, engine)

	liquidityObligation := num.NewUint(1230)
	// Limit orders don't provide enough liquidity
	limitOrders := []*types.Order{
		{
			Price:     num.NewUint(95),
			Size:      500,
			Remaining: 1,
			Side:      types.SideBuy,
		},
		{
			Price:     num.NewUint(97),
			Size:      1000,
			Remaining: 1,
			Side:      types.SideBuy,
		},
		{
			Price:     num.NewUint(104),
			Size:      500,
			Remaining: 1,
			Side:      types.SideSell,
		},
	}
	limitOrdersSuppliedLiquidity := num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(collateOrders(limitOrders, nil, nil), minLpPrice, maxLpPrice))
	require.True(t, limitOrdersSuppliedLiquidity.LT(liquidityObligation))

	engine.CalculateLiquidityImpliedVolumes(liquidityObligation, limitOrders, minLpPrice, maxLpPrice, buyShapes, sellShapes)

	var zero uint64
	require.Less(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Less(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Less(t, zero, validSell1.LiquidityImpliedVolume)
	require.Less(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity := num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(allOrders, minLpPrice, maxLpPrice))
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))
	require.True(t, totalSuppliedLiquidity.LT(liquidityObligation.Mul(liquidityObligation, num.NewUint(2))))

	// Limit buy orders provide enough liquidity
	limitOrders = []*types.Order{
		{
			Price:     num.NewUint(95),
			Size:      500,
			Remaining: 100,
			Side:      types.SideBuy,
		},
		{
			Price:     num.NewUint(97),
			Size:      1000,
			Remaining: 100,
			Side:      types.SideBuy,
		},
		{
			Price:     num.NewUint(104),
			Size:      500,
			Remaining: 1,
			Side:      types.SideSell,
		},
	}

	limitOrdersSuppliedLiquidity = num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(collateOrders(limitOrders, nil, nil), minLpPrice, maxLpPrice))
	require.True(t, limitOrdersSuppliedLiquidity.LT(liquidityObligation))

	engine.CalculateLiquidityImpliedVolumes(liquidityObligation, limitOrders, minLpPrice, maxLpPrice, buyShapes, sellShapes)

	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Less(t, zero, validSell1.LiquidityImpliedVolume)
	require.Less(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(allOrders, minLpPrice, maxLpPrice))
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))
	require.True(t, totalSuppliedLiquidity.LT(liquidityObligation.Mul(liquidityObligation, num.NewUint(2))))

	// Limit sell orders provide enough liquidity
	limitOrders = []*types.Order{
		{
			Price:     num.NewUint(95),
			Size:      500,
			Remaining: 1,
			Side:      types.SideBuy,
		},
		{
			Price:     num.NewUint(97),
			Size:      1000,
			Remaining: 1,
			Side:      types.SideBuy,
		},
		{
			Price:     num.NewUint(104),
			Size:      500,
			Remaining: 100,
			Side:      types.SideSell,
		},
	}
	limitOrdersSuppliedLiquidity = num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(allOrders, minLpPrice, maxLpPrice))
	require.True(t, limitOrdersSuppliedLiquidity.LT(liquidityObligation))

	engine.CalculateLiquidityImpliedVolumes(liquidityObligation, limitOrders, minLpPrice, maxLpPrice, buyShapes, sellShapes)

	require.Less(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Less(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(allOrders, minLpPrice, maxLpPrice))
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))
	tmp := liquidityObligation.Clone()
	tmp.Mul(tmp, num.NewUint(2))
	require.True(t, totalSuppliedLiquidity.LT(tmp))

	// Limit buy & sell orders provide enough liquidity
	limitOrders = []*types.Order{
		{
			Price:     num.NewUint(95),
			Size:      500,
			Remaining: 100,
			Side:      types.SideBuy,
		},
		{
			Price:     num.NewUint(97),
			Size:      1000,
			Remaining: 100,
			Side:      types.SideBuy,
		},
		{
			Price:     num.NewUint(104),
			Size:      500,
			Remaining: 100,
			Side:      types.SideSell,
		},
	}

	limitOrdersSuppliedLiquidity = num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(collateOrders(limitOrders, nil, nil), minLpPrice, maxLpPrice))
	require.True(t, limitOrdersSuppliedLiquidity.GT(liquidityObligation))

	engine.CalculateLiquidityImpliedVolumes(liquidityObligation, limitOrders, minLpPrice, maxLpPrice, buyShapes, sellShapes)

	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(allOrders, minLpPrice, maxLpPrice))
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))
}

func TestCalculateLiquidityImpliedSizes_NoValidOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minLpPrice := num.NewUint(89)
	maxLpPrice := num.NewUint(111)

	limitOrders := []*types.Order{}

	invalidBuy := &supplied.LiquidityOrder{
		Price:   num.UintZero().Sub(minLpPrice, num.NewUint(1)),
		Details: &types.LiquidityOrder{Proportion: 10},
	}
	buyShapes := []*supplied.LiquidityOrder{
		invalidBuy,
	}
	invalidSell := &supplied.LiquidityOrder{
		Price:   num.Sum(maxLpPrice, num.NewUint(1)),
		Details: &types.LiquidityOrder{Proportion: 33},
	}
	sellShapes := []*supplied.LiquidityOrder{
		invalidSell,
	}

	engine := supplied.NewEngine(riskModel, priceMonitor, "asset1", "market1", stubs.NewStateVar(), logging.NewTestLogger(), num.DecimalFromInt64(1))
	require.NotNil(t, engine)

	liquidityObligation := num.NewUint(20)
	engine.CalculateLiquidityImpliedVolumes(liquidityObligation, limitOrders, minLpPrice, maxLpPrice, buyShapes, sellShapes)

	// Expecting no failure with empty shapes
	engine.CalculateLiquidityImpliedVolumes(liquidityObligation, limitOrders, minLpPrice, maxLpPrice, []*supplied.LiquidityOrder{}, []*supplied.LiquidityOrder{})
}

func TestProbabilityOfTradingRecomputedAfterPriceRangeChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minLpPrice := num.NewUint(89)
	maxLpPrice := num.NewUint(111)

	order1 := &types.Order{
		Price:     minLpPrice,
		Size:      15,
		Remaining: 11,
		Side:      types.SideBuy,
	}
	order2 := &types.Order{
		Price:     maxLpPrice,
		Size:      60,
		Remaining: 60,
		Side:      types.SideSell,
	}

	orders := []*types.Order{
		order1,
		order2,
	}

	engine := supplied.NewEngine(riskModel, priceMonitor, "asset1", "market1", stubs.NewStateVar(), logging.NewTestLogger(), num.DecimalFromInt64(1))
	require.NotNil(t, engine)

	liquidity1 := num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(orders, minLpPrice, maxLpPrice))
	require.True(t, liquidity1.GT(num.NewUint(0)))

	liquidity2 := num.Min(engine.CalculateUnweightedBuySellLiquidityWithinLPRange(orders, minLpPrice, maxLpPrice))
	require.True(t, liquidity2.GT(num.NewUint(0)))
	require.Equal(t, liquidity1, liquidity2)
}

func collateOrders(limitOrders []*types.Order, buyShapes []*supplied.LiquidityOrder, sellShapes []*supplied.LiquidityOrder) []*types.Order {
	for _, s := range buyShapes {
		lo := &types.Order{
			Price:     s.Price,
			Size:      s.LiquidityImpliedVolume,
			Remaining: s.LiquidityImpliedVolume,
			Side:      types.SideBuy,
		}
		limitOrders = append(limitOrders, lo)
	}

	for _, s := range sellShapes {
		lo := &types.Order{
			Price:     s.Price,
			Size:      s.LiquidityImpliedVolume,
			Remaining: s.LiquidityImpliedVolume,
			Side:      types.SideSell,
		}
		limitOrders = append(limitOrders, lo)
	}
	return limitOrders
}

func min(d1, d2 num.Decimal) num.Decimal {
	if d1.LessThan(d2) {
		return d1
	}
	return d2
}
