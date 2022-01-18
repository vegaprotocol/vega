package supplied_test

import (
	"testing"

	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/liquidity/supplied"
	"code.vegaprotocol.io/vega/liquidity/supplied/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var (
	MarkPrice                          = num.NewUint(103)
	Horizon                            = num.DecimalFromFloat(0.001)
	DefaultInRangeProbabilityOfTrading = num.DecimalFromFloat(.5)
)

func TestCalculateSuppliedLiquidity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)

	minPrice := num.NewWrappedDecimal(num.NewUint(89), num.DecimalFromInt64(89))
	maxPrice := num.NewWrappedDecimal(num.NewUint(111), num.DecimalFromInt64(111))

	// No orders
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).AnyTimes()
	statevarEngine := stubs.NewStateVar()
	engine := supplied.NewEngine(riskModel, priceMonitor, "asset1", "market1", statevarEngine, logging.NewTestLogger())
	require.NotNil(t, engine)

	f := func() (*num.Uint, *num.Uint, error) { return MarkPrice.Clone(), MarkPrice.Clone(), nil }
	engine.SetGetStaticPricesFunc(f)

	liquidity := engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), []*types.Order{})
	require.Equal(t, num.NewUint(0), liquidity)

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

	riskModel.EXPECT().ProbabilityOfTrading(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(best *num.Uint, order *num.Uint, min num.Decimal, max num.Decimal, yFrac num.Decimal, isBid bool, applyMinMax bool) num.Decimal {
		if best.EQ(MarkPrice) && order.EQ(buyOrder1.Price) && isBid {
			return buyOrder1Prob
		}

		if best.EQ(MarkPrice) && order.EQ(sellOrder1.Price) && !isBid {
			return sellOrder1Prob
		}

		if best.EQ(MarkPrice) && order.EQ(sellOrder2.Price) && !isBid {
			return sellOrder2Prob
		}
		return num.DecimalZero()
	})

	statevarEngine.NewEvent("asset1", "market1", statevar.StateVarEventTypeAuctionEnded)
	liquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), []*types.Order{})
	require.Equal(t, num.NewUint(0), liquidity)

	// buyLiquidity := buyOrder1.Price.Float64() * float64(buyOrder1.Remaining) * buyOrder1Prob
	buyLiquidity := buyOrder1.Price.Clone()
	buyLiquidity = buyLiquidity.Mul(buyLiquidity, num.NewUint(buyOrder1.Remaining))
	bo1p := buyOrder1Prob.Mul(num.DecimalFromUint(buyLiquidity)).Mul(DefaultInRangeProbabilityOfTrading)
	buyLiquidity, _ = num.UintFromDecimal(bo1p)

	// sellLiquidity := sellOrder1.Price.Float64()*float64(sellOrder1.Remaining)*sellOrder1Prob + sellOrder2.Price.Float64()*float64(sellOrder2.Remaining)*sellOrder2Prob
	sellLiquidity1 := sellOrder1.Price.Clone()
	sellLiquidity1 = sellLiquidity1.Mul(sellLiquidity1, num.NewUint(sellOrder1.Remaining))
	so1 := sellOrder1Prob.Mul(num.DecimalFromUint(sellLiquidity1)).Mul(DefaultInRangeProbabilityOfTrading)

	sellLiquidity2 := sellOrder2.Price.Clone()
	sellLiquidity2 = sellLiquidity2.Mul(sellLiquidity2, num.NewUint(sellOrder2.Remaining))
	so2 := sellOrder2Prob.Mul(num.DecimalFromUint(sellLiquidity2)).Mul(DefaultInRangeProbabilityOfTrading)

	so := so1.Add(so2)
	sellLiquidity, _ := num.UintFromDecimal(so)
	expectedLiquidity := num.Min(buyLiquidity, sellLiquidity)
	liquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), []*types.Order{buyOrder1, sellOrder1, sellOrder2})
	require.Equal(t, expectedLiquidity, liquidity)

	// 2 buys, 2 sells
	buyOrder2 := &types.Order{
		Price:     num.NewUint(102),
		Size:      600,
		Remaining: 599,
		Side:      types.SideBuy,
	}
	buyOrder2Prob := 0.256

	//	buyLiquidity += buyOrder2.Price.Float64() * float64(buyOrder2.Remaining) * buyOrder2Prob
	buyLiquidity2 := buyOrder2.Price.Clone()
	buyLiquidity2 = buyLiquidity2.Mul(buyLiquidity2, num.NewUint(buyOrder2.Remaining))
	bo2 := num.DecimalFromFloat(buyOrder2Prob).Mul(num.DecimalFromUint(buyLiquidity2)).Mul(DefaultInRangeProbabilityOfTrading)
	buyLiquidity2, _ = num.UintFromDecimal(bo2)
	buyLiquidity = buyLiquidity.Add(buyLiquidity, buyLiquidity2)

	expectedLiquidity = num.Min(buyLiquidity, sellLiquidity)
	liquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), []*types.Order{buyOrder1, sellOrder1, sellOrder2, buyOrder2})
	require.Equal(t, expectedLiquidity, liquidity)
}

func Test_InteralConsistency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minPrice := num.NewWrappedDecimal(num.NewUint(89), num.DecimalFromInt64(89))
	maxPrice := num.NewWrappedDecimal(num.NewUint(111), num.DecimalFromInt64(111))
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).AnyTimes()

	limitOrders := []*types.Order{}

	buy := &supplied.LiquidityOrder{
		Price:      minPrice.Representation(),
		Proportion: 1,
	}
	buyShapes := []*supplied.LiquidityOrder{
		buy,
	}

	sell := &supplied.LiquidityOrder{
		Price:      maxPrice.Representation(),
		Proportion: 1,
	}

	sellShapes := []*supplied.LiquidityOrder{
		sell,
	}
	validBuy1Prob := num.DecimalFromFloat(0.1)
	validSell1Prob := num.DecimalFromFloat(0.22)

	riskModel.EXPECT().ProbabilityOfTrading(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(best *num.Uint, order *num.Uint, min num.Decimal, max num.Decimal, yFrac num.Decimal, isBid bool, applyMinMax bool) num.Decimal {
		if best.EQ(MarkPrice) && order.EQ(buy.Price) && isBid {
			return validBuy1Prob
		}

		if best.EQ(MarkPrice) && order.EQ(sell.Price) && !isBid {
			return validSell1Prob
		}
		return num.DecimalZero()
	})

	statevarEngine := stubs.NewStateVar()
	engine := supplied.NewEngine(riskModel, priceMonitor, "asset1", "market1", statevarEngine, logging.NewTestLogger())
	require.NotNil(t, engine)
	f := func() (*num.Uint, *num.Uint, error) { return MarkPrice.Clone(), MarkPrice.Clone(), nil }
	engine.SetGetStaticPricesFunc(f)
	statevarEngine.NewEvent("asset1", "market1", statevar.StateVarEventTypeAuctionEnded)

	// Negative liquidity obligation -> 0 sizes on all orders
	liquidityObligation := num.NewUint(100)
	err := engine.CalculateLiquidityImpliedVolumes(MarkPrice.Clone(), MarkPrice.Clone(), liquidityObligation.Clone(), limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	var zero uint64 = 0
	require.Less(t, zero, buy.LiquidityImpliedVolume)
	require.Less(t, zero, sell.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity := engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), allOrders)
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))
}

func TestCalculateLiquidityImpliedSizes_NoLimitOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minPrice := num.NewWrappedDecimal(num.NewUint(89), num.DecimalFromInt64(89))
	maxPrice := num.NewWrappedDecimal(num.NewUint(111), num.DecimalFromInt64(111))
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).AnyTimes()

	limitOrders := []*types.Order{}

	validBuy1 := &supplied.LiquidityOrder{
		Price:      minPrice.Representation(),
		Proportion: 20,
	}

	validBuy2 := &supplied.LiquidityOrder{
		Price:      num.Sum(minPrice.Representation(), num.NewUint(1)),
		Proportion: 30,
	}
	buyShapes := []*supplied.LiquidityOrder{
		validBuy1,
		validBuy2,
	}
	validSell1 := &supplied.LiquidityOrder{
		Price:      num.Zero().Sub(maxPrice.Representation(), num.NewUint(1)),
		Proportion: 11,
	}
	validSell2 := &supplied.LiquidityOrder{
		Price:      maxPrice.Representation(),
		Proportion: 22,
	}
	sellShapes := []*supplied.LiquidityOrder{
		validSell1,
		validSell2,
	}
	validBuy1Prob := num.DecimalFromFloat(0.1)
	validBuy2Prob := num.DecimalFromFloat(0.2)
	validSell1Prob := num.DecimalFromFloat(0.22)
	validSell2Prob := num.DecimalFromFloat(0.11)

	riskModel.EXPECT().ProbabilityOfTrading(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(best *num.Uint, order *num.Uint, min num.Decimal, max num.Decimal, yFrac num.Decimal, isBid bool, applyMinMax bool) num.Decimal {
		if best.EQ(MarkPrice) && order.EQ(validBuy1.Price) && isBid {
			return validBuy1Prob
		}
		if best.EQ(MarkPrice) && order.EQ(validBuy2.Price) && isBid {
			return validBuy2Prob
		}
		if best.EQ(MarkPrice) && order.EQ(validSell1.Price) && !isBid {
			return validSell1Prob
		}
		if best.EQ(MarkPrice) && order.EQ(validSell2.Price) && !isBid {
			return validSell2Prob
		}
		return num.DecimalZero()
	})

	statevarEngine := stubs.NewStateVar()
	engine := supplied.NewEngine(riskModel, priceMonitor, "asset1", "market1", statevarEngine, logging.NewTestLogger())
	require.NotNil(t, engine)
	f := func() (*num.Uint, *num.Uint, error) { return MarkPrice.Clone(), MarkPrice.Clone(), nil }
	engine.SetGetStaticPricesFunc(f)
	statevarEngine.NewEvent("asset1", "market1", statevar.StateVarEventTypeAuctionEnded)

	// No liquidity obligation -> 0 sizes on all orders
	liquidityObligation := num.NewUint(0)
	err := engine.CalculateLiquidityImpliedVolumes(MarkPrice.Clone(), MarkPrice.Clone(), liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	var zero uint64 = 0
	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity := engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), allOrders)
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))

	// 0 liquidity obligation -> 0 sizes on all orders
	liquidityObligation = num.NewUint(0)
	err = engine.CalculateLiquidityImpliedVolumes(MarkPrice.Clone(), MarkPrice.Clone(), liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), allOrders)
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))

	// Positive liquidity obligation -> positive sizes on orders -> suplied liquidity >= liquidity obligation
	liquidityObligation = num.NewUint(25)
	err = engine.CalculateLiquidityImpliedVolumes(MarkPrice.Clone(), MarkPrice.Clone(), liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	loDec := liquidityObligation.ToDecimal()
	vb1Prop := num.DecimalFromFloat(float64(validBuy1.Proportion))
	vb2Prop := num.DecimalFromFloat(float64(validBuy2.Proportion))
	vs1Prop := num.DecimalFromFloat(float64(validSell1.Proportion))
	vs2Prop := num.DecimalFromFloat(float64(validSell2.Proportion))
	vb1Price := validBuy1.Price.ToDecimal()
	vb2Price := validBuy2.Price.ToDecimal()
	vs1Price := validSell1.Price.ToDecimal()
	vs2Price := validSell2.Price.ToDecimal()
	expVolVB1 := loDec.Mul(vb1Prop).Div(vb1Prop.Add(vb2Prop)).Div(validBuy1Prob).Div(DefaultInRangeProbabilityOfTrading).Div(vb1Price).Ceil()
	expVolVB2 := loDec.Mul(vb2Prop).Div(vb1Prop.Add(vb2Prop)).Div(validBuy2Prob).Div(DefaultInRangeProbabilityOfTrading).Div(vb2Price).Ceil()

	expVolVS1 := loDec.Mul(vs1Prop).Div(vs1Prop.Add(vs2Prop)).Div(validSell1Prob).Div(DefaultInRangeProbabilityOfTrading).Div(vs1Price).Ceil()
	expVolVS2 := loDec.Mul(vs2Prop).Div(vs1Prop.Add(vs2Prop)).Div(validSell2Prob).Div(DefaultInRangeProbabilityOfTrading).Div(vs2Price).Ceil()

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
	totalSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), allOrders)
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))
	require.True(t, totalSuppliedLiquidity.LT(liquidityObligation.Mul(liquidityObligation, num.NewUint(2))))
}

func TestCalculateLiquidityImpliedSizes_WithLimitOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minPrice := num.NewWrappedDecimal(num.NewUint(89), num.DecimalFromInt64(89))
	maxPrice := num.NewWrappedDecimal(num.NewUint(111), num.DecimalFromInt64(111))
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).AnyTimes()

	validBuy1 := &supplied.LiquidityOrder{
		Price:      minPrice.Representation(),
		Proportion: 20,
	}
	validBuy2 := &supplied.LiquidityOrder{
		Price:      num.Sum(minPrice.Representation(), num.NewUint(1)),
		Proportion: 30,
	}
	buyShapes := []*supplied.LiquidityOrder{
		validBuy1,
		validBuy2,
	}
	validSell1 := &supplied.LiquidityOrder{
		Price:      num.Zero().Sub(maxPrice.Representation(), num.NewUint(1)),
		Proportion: 11,
	}
	validSell2 := &supplied.LiquidityOrder{
		Price:      maxPrice.Representation(),
		Proportion: 22,
	}
	sellShapes := []*supplied.LiquidityOrder{
		validSell1,
		validSell2,
	}

	statevarEngine := stubs.NewStateVar()
	engine := supplied.NewEngine(riskModel, priceMonitor, "asset1", "market1", statevarEngine, logging.NewTestLogger())
	require.NotNil(t, engine)
	f := func() (*num.Uint, *num.Uint, error) { return MarkPrice.Clone(), MarkPrice.Clone(), nil }
	engine.SetGetStaticPricesFunc(f)

	liquidityObligation := num.NewUint(123) // Was 123.45
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

	riskModel.EXPECT().ProbabilityOfTrading(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(best *num.Uint, order *num.Uint, min num.Decimal, max num.Decimal, yFrac num.Decimal, isBid bool, applyMinMax bool) num.Decimal {
		if best.EQ(MarkPrice) && order.EQ(limitOrders[0].Price) && isBid {
			return num.DecimalFromFloat(0.175)
		}
		if best.EQ(MarkPrice) && order.EQ(limitOrders[1].Price) && isBid {
			return num.DecimalFromFloat(0.312)
		}
		if best.EQ(MarkPrice) && order.EQ(limitOrders[2].Price) && !isBid {
			return num.DecimalFromFloat(0.5)
		}

		if best.EQ(MarkPrice) && order.EQ(validBuy1.Price) && isBid {
			return num.DecimalFromFloat(0.1)
		}
		if best.EQ(MarkPrice) && order.EQ(validBuy2.Price) && isBid {
			return num.DecimalFromFloat(0.1)
		}
		if best.EQ(MarkPrice) && order.EQ(validSell1.Price) && isBid {
			return num.DecimalFromFloat(0.22)
		}
		if best.EQ(MarkPrice) && order.EQ(validSell2.Price) && isBid {
			return num.DecimalFromFloat(0.11)
		}
		return num.DecimalZero()
	})
	statevarEngine.NewEvent("asset1", "market1", statevar.StateVarEventTypeAuctionEnded)

	limitOrdersSuppliedLiquidity := engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), collateOrders(limitOrders, nil, nil))
	require.True(t, limitOrdersSuppliedLiquidity.LT(liquidityObligation))

	err := engine.CalculateLiquidityImpliedVolumes(MarkPrice.Clone(), MarkPrice.Clone(), liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	var zero uint64 = 0
	require.Less(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Less(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Less(t, zero, validSell1.LiquidityImpliedVolume)
	require.Less(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity := engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), allOrders)
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

	limitOrdersSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), collateOrders(limitOrders, nil, nil))
	require.True(t, limitOrdersSuppliedLiquidity.LT(liquidityObligation))

	err = engine.CalculateLiquidityImpliedVolumes(MarkPrice.Clone(), MarkPrice.Clone(), liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Less(t, zero, validSell1.LiquidityImpliedVolume)
	require.Less(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), allOrders)
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

	limitOrdersSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), collateOrders(limitOrders, nil, nil))
	require.True(t, limitOrdersSuppliedLiquidity.LT(liquidityObligation))

	err = engine.CalculateLiquidityImpliedVolumes(MarkPrice.Clone(), MarkPrice.Clone(), liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Less(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Less(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, allOrders)
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

	limitOrdersSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), collateOrders(limitOrders, nil, nil))
	require.True(t, limitOrdersSuppliedLiquidity.GT(liquidityObligation))

	err = engine.CalculateLiquidityImpliedVolumes(MarkPrice.Clone(), MarkPrice.Clone(), liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), allOrders)
	require.True(t, totalSuppliedLiquidity.GTE(liquidityObligation))
}

func TestCalculateLiquidityImpliedSizes_NoValidOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minPrice := num.NewWrappedDecimal(num.NewUint(89), num.DecimalFromInt64(89))
	maxPrice := num.NewWrappedDecimal(num.NewUint(111), num.DecimalFromInt64(111))
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).AnyTimes()

	limitOrders := []*types.Order{}

	invalidBuy := &supplied.LiquidityOrder{
		Price:      num.Zero().Sub(minPrice.Representation(), num.NewUint(1)),
		Proportion: 10,
	}
	buyShapes := []*supplied.LiquidityOrder{
		invalidBuy,
	}
	invalidSell := &supplied.LiquidityOrder{
		Price:      num.Sum(maxPrice.Representation(), num.NewUint(1)),
		Proportion: 33,
	}
	sellShapes := []*supplied.LiquidityOrder{
		invalidSell,
	}

	riskModel.EXPECT().ProbabilityOfTrading(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(best *num.Uint, order *num.Uint, min num.Decimal, max num.Decimal, yFrac num.Decimal, isBid bool, applyMinMax bool) num.Decimal {
		if best.EQ(MarkPrice) && order.EQ(invalidBuy.Price) && isBid {
			return num.DecimalZero()
		}

		if best.EQ(MarkPrice) && order.EQ(invalidSell.Price) && !isBid {
			return num.DecimalZero()
		}
		return num.DecimalFromFloat(0.1)
	})

	statevarEngine := stubs.NewStateVar()
	engine := supplied.NewEngine(riskModel, priceMonitor, "asset1", "market1", statevarEngine, logging.NewTestLogger())
	require.NotNil(t, engine)
	f := func() (*num.Uint, *num.Uint, error) { return MarkPrice.Clone(), MarkPrice.Clone(), nil }
	engine.SetGetStaticPricesFunc(f)
	statevarEngine.NewEvent("asset1", "market1", statevar.StateVarEventTypeAuctionEnded)

	liquidityObligation := num.NewUint(20)
	// Expecting no error now (other component assures orders get shifted to valid price range, failsafe in place to safeguard against near-zero probability of trading)
	err := engine.CalculateLiquidityImpliedVolumes(MarkPrice.Clone(), MarkPrice.Clone(), liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	// We do expect an error when no orders specified though.
	err = engine.CalculateLiquidityImpliedVolumes(MarkPrice.Clone(), MarkPrice.Clone(), liquidityObligation, limitOrders, []*supplied.LiquidityOrder{}, []*supplied.LiquidityOrder{})
	require.Error(t, err)
}

func TestProbabilityOfTradingRecomputedAfterPriceRangeChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minPrice := num.NewWrappedDecimal(num.NewUint(89), num.DecimalFromInt64(89))
	maxPrice := num.NewWrappedDecimal(num.NewUint(111), num.DecimalFromInt64(111))

	order1 := &types.Order{
		Price:     minPrice.Representation(),
		Size:      15,
		Remaining: 11,
		Side:      types.SideBuy,
	}
	order2 := &types.Order{
		Price:     maxPrice.Representation(),
		Size:      60,
		Remaining: 60,
		Side:      types.SideSell,
	}

	orders := []*types.Order{
		order1,
		order2,
	}

	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).AnyTimes()
	riskModel.EXPECT().ProbabilityOfTrading(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(best *num.Uint, order *num.Uint, min num.Decimal, max num.Decimal, yFrac num.Decimal, isBid bool, applyMinMax bool) num.Decimal {
		if best.EQ(MarkPrice) && order.EQ(order1.Price) && isBid {
			return num.DecimalFromFloat(0.123)
		}

		if best.EQ(MarkPrice) && order.EQ(order2.Price) && !isBid {
			return num.DecimalFromFloat(0.234)
		}
		return num.DecimalZero()
	})
	statevarEngine := stubs.NewStateVar()
	engine := supplied.NewEngine(riskModel, priceMonitor, "asset1", "market1", statevarEngine, logging.NewTestLogger())
	require.NotNil(t, engine)
	f := func() (*num.Uint, *num.Uint, error) { return MarkPrice.Clone(), MarkPrice.Clone(), nil }
	engine.SetGetStaticPricesFunc(f)
	statevarEngine.NewEvent("asset1", "market1", statevar.StateVarEventTypeAuctionEnded)
	liquidity1 := engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), orders)
	require.True(t, liquidity1.GT(num.NewUint(0)))

	liquidity2 := engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), orders)
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
