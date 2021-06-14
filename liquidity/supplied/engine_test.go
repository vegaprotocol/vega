package supplied_test

import (
	"math"
	"testing"

	"code.vegaprotocol.io/vega/liquidity/supplied"
	"code.vegaprotocol.io/vega/liquidity/supplied/mocks"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var (
	MarkPrice = num.NewUint(123)
	Horizon   = num.DecimalFromFloat(0.001)
)

func TestCalculateSuppliedLiquidity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)

	// Prices are integers but for now the functions take float64 TODO UINT
	minPrice := num.NewUint(89)
	maxPrice := num.NewUint(111)

	// No orders
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(1)

	engine := supplied.NewEngine(riskModel, priceMonitor)
	require.NotNil(t, engine)

	liquidity := engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), []*types.Order{})
	require.Equal(t, num.NewUint(0), liquidity)

	// 1 buy, no sells
	buyOrder1 := &types.Order{
		Price:     num.NewUint(102),
		Size:      30,
		Remaining: 25,
		Side:      types.Side_SIDE_BUY,
	}

	buyOrder1Prob := num.DecimalFromFloat(0.256)
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Clone(), buyOrder1.Price.Clone(), minPrice.Clone(), maxPrice.Clone(), Horizon, true, true).Return(buyOrder1Prob).Times(4)

	liquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), []*types.Order{buyOrder1})
	require.Equal(t, num.NewUint(0), liquidity)

	// 1 buy, 2 sells
	sellOrder1 := &types.Order{
		Price:     num.NewUint(99),
		Size:      15,
		Remaining: 11,
		Side:      types.Side_SIDE_SELL,
	}
	sellOrder2 := &types.Order{
		Price:     num.NewUint(97),
		Size:      60,
		Remaining: 60,
		Side:      types.Side_SIDE_SELL,
	}

	sellOrder1Prob := num.DecimalFromFloat(0.33)
	sellOrder2Prob := num.DecimalFromFloat(0.17)
	// buyLiquidity := buyOrder1.Price.Float64() * float64(buyOrder1.Remaining) * buyOrder1Prob
	buyLiquidity := buyOrder1.Price.Clone()
	buyLiquidity = buyLiquidity.Mul(buyLiquidity, num.NewUint(buyOrder1.Remaining))
	bo1p := buyOrder1Prob.Mul(num.DecimalFromUint(buyLiquidity))
	buyLiquidity, _ = num.UintFromDecimal(bo1p)

	// sellLiquidity := sellOrder1.Price.Float64()*float64(sellOrder1.Remaining)*sellOrder1Prob + sellOrder2.Price.Float64()*float64(sellOrder2.Remaining)*sellOrder2Prob
	sellLiquidity1 := sellOrder1.Price.Clone()
	sellLiquidity1 = sellLiquidity1.Mul(sellLiquidity1, num.NewUint(sellOrder1.Remaining))
	so1 := sellOrder1Prob.Mul(num.DecimalFromUint(sellLiquidity1))

	sellLiquidity2 := sellOrder2.Price.Clone()
	sellLiquidity2 = sellLiquidity2.Mul(sellLiquidity2, num.NewUint(sellOrder2.Remaining))
	so2 := sellOrder2Prob.Mul(num.DecimalFromUint(sellLiquidity2))

	so := so1.Add(so2)
	sellLiquidity, _ := num.UintFromDecimal(so)

	expectedLiquidity := num.Min(buyLiquidity, sellLiquidity)

	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Clone(), sellOrder1.Price.Clone(), minPrice.Clone(), maxPrice.Clone(), Horizon, false, true).Return(sellOrder1Prob).Times(2)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Clone(), sellOrder2.Price.Clone(), minPrice.Clone(), maxPrice.Clone(), Horizon, false, true).Return(sellOrder2Prob).Times(2)

	liquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), []*types.Order{buyOrder1, sellOrder1, sellOrder2})
	require.Equal(t, expectedLiquidity, liquidity)

	// 2 buys, 2 sells
	buyOrder2 := &types.Order{
		Price:     num.NewUint(102),
		Size:      600,
		Remaining: 599,
		Side:      types.Side_SIDE_BUY,
	}
	buyOrder2Prob := 0.256

	//	buyLiquidity += buyOrder2.Price.Float64() * float64(buyOrder2.Remaining) * buyOrder2Prob
	buyLiquidity2 := buyOrder2.Price.Clone()
	buyLiquidity2 = buyLiquidity2.Mul(buyLiquidity2, num.NewUint(buyOrder2.Remaining))
	bo2 := num.DecimalFromFloat(buyOrder2Prob).Mul(num.DecimalFromUint(buyLiquidity2))
	buyLiquidity2, _ = num.UintFromDecimal(bo2)
	buyLiquidity = buyLiquidity.Add(buyLiquidity, buyLiquidity2)

	expectedLiquidity = num.Min(buyLiquidity, sellLiquidity)

	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice)

	liquidity = engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), []*types.Order{buyOrder1, sellOrder1, sellOrder2, buyOrder2})
	require.Equal(t, expectedLiquidity, liquidity)
}

func Test_InteralConsistency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minPrice := num.NewUint(89)
	maxPrice := num.NewUint(111)
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(2)

	limitOrders := []*types.Order{}

	buy := &supplied.LiquidityOrder{
		Price:      minPrice,
		Proportion: 1,
	}
	buyShapes := []*supplied.LiquidityOrder{
		buy,
	}

	sell := &supplied.LiquidityOrder{
		Price:      maxPrice,
		Proportion: 1,
	}

	sellShapes := []*supplied.LiquidityOrder{
		sell,
	}
	validBuy1Prob := num.DecimalFromFloat(0.1)
	validSell1Prob := num.DecimalFromFloat(0.22)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice, buy.Price, minPrice, maxPrice, Horizon, true, true).Return(validBuy1Prob).Times(2)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice, sell.Price, minPrice, maxPrice, Horizon, false, true).Return(validSell1Prob).Times(2)

	engine := supplied.NewEngine(riskModel, priceMonitor)
	require.NotNil(t, engine)

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
	minPrice := num.NewUint(89)
	maxPrice := num.NewUint(111)
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(6)

	limitOrders := []*types.Order{}

	validBuy1 := &supplied.LiquidityOrder{
		Price:      minPrice,
		Proportion: 20,
	}

	validBuy2 := &supplied.LiquidityOrder{
		Price:      num.NewUint(0).Add(minPrice, num.NewUint(1)),
		Proportion: 30,
	}
	buyShapes := []*supplied.LiquidityOrder{
		validBuy1,
		validBuy2,
	}
	validSell1 := &supplied.LiquidityOrder{
		Price:      num.NewUint(0).Sub(maxPrice, num.NewUint(1)),
		Proportion: 11,
	}
	validSell2 := &supplied.LiquidityOrder{
		Price:      maxPrice,
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
	riskModel.EXPECT().ProbabilityOfTrading(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), Horizon, true, true).MinTimes(2).MaxTimes(10).DoAndReturn(func(mPrice, orderPrice, min, max *num.Uint, _ num.Decimal, _, _ bool) num.Decimal {
		require.True(t, mPrice.EQ(MarkPrice))
		require.True(t, min.EQ(minPrice))
		require.True(t, max.EQ(maxPrice))
		if orderPrice.EQ(validBuy1.Price) {
			return validBuy1Prob
		}
		if orderPrice.EQ(validBuy2.Price) {
			return validBuy2Prob
		}
		require.True(t, false, "given order price unknown: %d", orderPrice.Uint64())
		return num.DecimalFromFloat(0)
	})
	riskModel.EXPECT().ProbabilityOfTrading(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), Horizon, false, true).MinTimes(2).MaxTimes(11).DoAndReturn(func(mPrice, orderPrice, min, max *num.Uint, _ num.Decimal, _, _ bool) num.Decimal {
		require.True(t, mPrice.EQ(MarkPrice))
		require.True(t, min.EQ(minPrice))
		require.True(t, max.EQ(maxPrice))
		if orderPrice.EQ(validSell1.Price) {
			return validSell1Prob
		}
		if orderPrice.EQ(validSell2.Price) {
			return validSell2Prob
		}
		require.True(t, false, "given order price unknown: %d", orderPrice.Uint64())
		return num.DecimalFromFloat(0)
	})

	engine := supplied.NewEngine(riskModel, priceMonitor)
	require.NotNil(t, engine)

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
	expVolVB1 := loDec.Mul(vb1Prop).Div(vb1Prop.Add(vb2Prop)).Div(validBuy1Prob).Div(vb1Price).Ceil()
	expVolVB2 := loDec.Mul(vb2Prop).Div(vb1Prop.Add(vb2Prop)).Div(validBuy2Prob).Div(vb2Price).Ceil()

	expVolVS1 := loDec.Mul(vs1Prop).Div(vs1Prop.Add(vs2Prop)).Div(validSell1Prob).Div(vs1Price).Ceil()
	expVolVS2 := loDec.Mul(vs2Prop).Div(vs1Prop.Add(vs2Prop)).Div(validSell2Prob).Div(vs2Price).Ceil()

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
	minPrice := num.NewUint(89)
	maxPrice := num.NewUint(111)
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(12)

	validBuy1 := &supplied.LiquidityOrder{
		Price:      minPrice,
		Proportion: 20,
	}
	validBuy2 := &supplied.LiquidityOrder{
		Price:      num.NewUint(0).Add(minPrice, num.NewUint(1)),
		Proportion: 30,
	}
	buyShapes := []*supplied.LiquidityOrder{
		validBuy1,
		validBuy2,
	}
	validSell1 := &supplied.LiquidityOrder{
		Price:      num.NewUint(0).Sub(maxPrice, num.NewUint(1)),
		Proportion: 11,
	}
	validSell2 := &supplied.LiquidityOrder{
		Price:      maxPrice,
		Proportion: 22,
	}
	sellShapes := []*supplied.LiquidityOrder{
		validSell1,
		validSell2,
	}

	riskModel.EXPECT().ProbabilityOfTrading(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), Horizon, true, true).MinTimes(2).MaxTimes(10).DoAndReturn(func(mPrice, oPrice, min, max *num.Uint, _ num.Decimal, _, _ bool) num.Decimal {
		require.True(t, mPrice.EQ(MarkPrice))
		require.True(t, min.EQ(minPrice))
		require.True(t, max.EQ(maxPrice))
		if oPrice.EQ(validBuy1.Price) {
			return num.DecimalFromFloat(.1)
		}
		if oPrice.EQ(validBuy2.Price) {
			return num.DecimalFromFloat(.2)
		}
		require.Equal(t, "", oPrice.String(), "invalid price")
		return num.DecimalFromFloat(0)
	})
	riskModel.EXPECT().ProbabilityOfTrading(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), Horizon, false, true).MinTimes(2).MaxTimes(10).DoAndReturn(func(mPrice, oPrice, min, max *num.Uint, _ num.Decimal, _, _ bool) num.Decimal {
		require.True(t, mPrice.EQ(MarkPrice))
		require.True(t, min.EQ(minPrice))
		require.True(t, max.EQ(maxPrice))
		if oPrice.EQ(validSell1.Price) {
			return num.DecimalFromFloat(.22)
		}
		if oPrice.EQ(validSell2.Price) {
			return num.DecimalFromFloat(.11)
		}
		require.Equal(t, "", oPrice.String(), "invalid price")
		return num.DecimalFromFloat(0)
	})

	engine := supplied.NewEngine(riskModel, priceMonitor)
	require.NotNil(t, engine)

	liquidityObligation := num.NewUint(123) // Was 123.45
	// Limit orders don't provide enough liquidity
	limitOrders := []*types.Order{
		{
			Price:     num.NewUint(95),
			Size:      500,
			Remaining: 1,
			Side:      types.Side_SIDE_BUY,
		},
		{
			Price:     num.NewUint(97),
			Size:      1000,
			Remaining: 1,
			Side:      types.Side_SIDE_BUY,
		},
		{
			Price:     num.NewUint(104),
			Size:      500,
			Remaining: 1,
			Side:      types.Side_SIDE_SELL,
		},
	}

	riskModel.EXPECT().ProbabilityOfTrading(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), Horizon, gomock.Any(), true).MinTimes(3).MaxTimes(15).DoAndReturn(func(mPrice, oPrice, min, max *num.Uint, _ num.Decimal, sell, _ bool) num.Decimal {
		require.True(t, mPrice.EQ(MarkPrice))
		require.True(t, min.EQ(minPrice))
		require.True(t, max.EQ(maxPrice))
		if sell {
			require.True(t, oPrice.EQ(limitOrders[2].Price))
			return num.DecimalFromFloat(.5)
		}
		if oPrice.EQ(limitOrders[0].Price) {
			return num.DecimalFromFloat(.175)
		}
		if oPrice.EQ(limitOrders[1].Price) {
			return num.DecimalFromFloat(.312)
		}
		require.Equal(t, "wrong price", oPrice.String())
		return num.DecimalFromFloat(0)
	})

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
			Side:      types.Side_SIDE_BUY,
		},
		{
			Price:     num.NewUint(97),
			Size:      1000,
			Remaining: 100,
			Side:      types.Side_SIDE_BUY,
		},
		{
			Price:     num.NewUint(104),
			Size:      500,
			Remaining: 1,
			Side:      types.Side_SIDE_SELL,
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

	//Limit sell orders provide enough liquidity
	limitOrders = []*types.Order{
		{
			Price:     num.NewUint(95),
			Size:      500,
			Remaining: 1,
			Side:      types.Side_SIDE_BUY,
		},
		{
			Price:     num.NewUint(97),
			Size:      1000,
			Remaining: 1,
			Side:      types.Side_SIDE_BUY,
		},
		{
			Price:     num.NewUint(104),
			Size:      500,
			Remaining: 100,
			Side:      types.Side_SIDE_SELL,
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
			Side:      types.Side_SIDE_BUY,
		},
		{
			Price:     num.NewUint(97),
			Size:      1000,
			Remaining: 100,
			Side:      types.Side_SIDE_BUY,
		},
		{
			Price:     num.NewUint(104),
			Size:      500,
			Remaining: 100,
			Side:      types.Side_SIDE_SELL,
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
	minPrice := num.NewUint(89)
	maxPrice := num.NewUint(111)
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(2)

	limitOrders := []*types.Order{}

	invalidBuy := &supplied.LiquidityOrder{
		Price:      num.NewUint(0).Sub(minPrice, num.NewUint(1)),
		Proportion: 10,
	}
	buyShapes := []*supplied.LiquidityOrder{
		invalidBuy,
	}
	invalidSell := &supplied.LiquidityOrder{
		Price:      num.NewUint(0).Add(maxPrice, num.NewUint(1)),
		Proportion: 33,
	}
	sellShapes := []*supplied.LiquidityOrder{
		invalidSell,
	}
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice, invalidBuy.Price, minPrice, maxPrice, Horizon, true, true).Return(0.0).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice, invalidSell.Price, minPrice, maxPrice, Horizon, false, true).Return(0.0).Times(1)

	engine := supplied.NewEngine(riskModel, priceMonitor)
	require.NotNil(t, engine)

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
	minPrice := 89.2
	maxPrice := 111.1

	var minPriceInt = uint64(math.Ceil(minPrice))
	var maxPriceInt = uint64(math.Floor(maxPrice))

	order1 := &types.Order{
		Price:     num.NewUint(minPriceInt),
		Size:      15,
		Remaining: 11,
		Side:      types.Side_SIDE_BUY,
	}
	order2 := &types.Order{
		Price:     num.NewUint(maxPriceInt),
		Size:      60,
		Remaining: 60,
		Side:      types.Side_SIDE_SELL,
	}

	orders := []*types.Order{
		order1,
		order2,
	}

	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice, order1.Price, minPriceInt, maxPriceInt, Horizon, true, true).Return(0.123).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice, order2.Price, minPrice, maxPrice, Horizon, false, true).Return(0.234).Times(1)

	engine := supplied.NewEngine(riskModel, priceMonitor)
	require.NotNil(t, engine)

	liquidity1 := engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), orders)
	require.Less(t, 0.0, liquidity1)

	// Change minPrice, maxPrice and verify that probability of trading is called with new values
	minPrice -= 10
	maxPrice += 10
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice, order1.Price, minPrice, maxPrice, Horizon, true, true).Return(0.123).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice, order2.Price, minPrice, maxPrice, Horizon, false, true).Return(0.234).Times(1)

	liquidity2 := engine.CalculateSuppliedLiquidity(MarkPrice.Clone(), MarkPrice.Clone(), orders)
	require.Less(t, 0.0, liquidity2)
	require.Equal(t, liquidity1, liquidity2)

}

func collateOrders(limitOrders []*types.Order, buyShapes []*supplied.LiquidityOrder, sellShapes []*supplied.LiquidityOrder) []*types.Order {
	for _, s := range buyShapes {
		lo := &types.Order{
			Price:     s.Price,
			Size:      s.LiquidityImpliedVolume,
			Remaining: s.LiquidityImpliedVolume,
			Side:      types.Side_SIDE_BUY,
		}
		limitOrders = append(limitOrders, lo)
	}

	for _, s := range sellShapes {
		lo := &types.Order{
			Price:     s.Price,
			Size:      s.LiquidityImpliedVolume,
			Remaining: s.LiquidityImpliedVolume,
			Side:      types.Side_SIDE_SELL,
		}
		limitOrders = append(limitOrders, lo)
	}
	return limitOrders
}
