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

const (
	Horizon = 0.001
)

var (
	MarkPrice = num.NewUint(123)
)

func TestCalculateSuppliedLiquidity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)

	minPrice := 89.2
	maxPrice := 111.1

	// No orders
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(1)

	engine := supplied.NewEngine(riskModel, priceMonitor)
	require.NotNil(t, engine)

	liquidity := engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, []*types.Order{})
	require.Equal(t, 0.0, liquidity)

	// 1 buy, no sells
	buyOrder1 := &types.Order{
		Price:     num.NewUint(102),
		Size:      30,
		Remaining: 25,
		Side:      types.Side_SIDE_BUY,
	}

	buyOrder1Prob := 0.256
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, buyOrder1.Price.Float64(), true, true, minPrice, maxPrice).Return(buyOrder1Prob).Times(1)

	liquidity = engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, []*types.Order{buyOrder1})
	require.Equal(t, 0.0, liquidity)

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

	sellOrder1Prob := 0.33
	sellOrder2Prob := 0.17
	buyLiquidity := buyOrder1.Price.Float64() * float64(buyOrder1.Remaining) * buyOrder1Prob
	sellLiquidity := sellOrder1.Price.Float64()*float64(sellOrder1.Remaining)*sellOrder1Prob + sellOrder2.Price.Float64()*float64(sellOrder2.Remaining)*sellOrder2Prob
	expectedLiquidity := math.Min(buyLiquidity, sellLiquidity)

	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, sellOrder1.Price.Float64(), false, true, minPrice, maxPrice).Return(sellOrder1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, sellOrder2.Price.Float64(), false, true, minPrice, maxPrice).Return(sellOrder2Prob).Times(1)

	liquidity = engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, []*types.Order{buyOrder1, sellOrder1, sellOrder2})
	require.InDelta(t, expectedLiquidity, liquidity, 0.01)

	// 2 buys, 2 sells
	buyOrder2 := &types.Order{
		Price:     num.NewUint(102),
		Size:      600,
		Remaining: 599,
		Side:      types.Side_SIDE_BUY,
	}

	buyLiquidity += buyOrder2.Price.Float64() * float64(buyOrder2.Remaining) * buyOrder1Prob
	expectedLiquidity = math.Min(buyLiquidity, sellLiquidity)

	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice)

	liquidity = engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, []*types.Order{buyOrder1, sellOrder1, sellOrder2, buyOrder2})
	require.InDelta(t, expectedLiquidity, liquidity, 0.01)
}

func Test_InteralConsistency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minPrice := 89.2
	maxPrice := 111.1
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(2)

	limitOrders := []*types.Order{}

	var minPriceInt = uint64(math.Ceil(minPrice))
	var maxPriceInt = uint64(math.Floor(maxPrice))

	buy := &supplied.LiquidityOrder{
		Price:      num.NewUint(minPriceInt),
		Proportion: 1,
	}
	buyShapes := []*supplied.LiquidityOrder{
		buy,
	}

	sell := &supplied.LiquidityOrder{
		Price:      num.NewUint(maxPriceInt),
		Proportion: 1,
	}

	sellShapes := []*supplied.LiquidityOrder{
		sell,
	}
	validBuy1Prob := 0.1
	validSell1Prob := 0.22
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, buy.Price.Float64(), true, true, minPrice, maxPrice).Return(validBuy1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, sell.Price.Float64(), false, true, minPrice, maxPrice).Return(validSell1Prob).Times(1)

	engine := supplied.NewEngine(riskModel, priceMonitor)
	require.NotNil(t, engine)

	// Negative liquidity obligation -> 0 sizes on all orders
	liquidityObligation := 100.0
	err := engine.CalculateLiquidityImpliedVolumes(MarkPrice, MarkPrice, liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	var zero uint64 = 0
	require.Less(t, zero, buy.LiquidityImpliedVolume)
	require.Less(t, zero, sell.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity := engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, allOrders)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)
}

func TestCalculateLiquidityImpliedSizes_NoLimitOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minPrice := 89.2
	maxPrice := 111.1
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(5)

	limitOrders := []*types.Order{}

	var minPriceInt = uint64(math.Ceil(minPrice))
	var maxPriceInt = uint64(math.Floor(maxPrice))

	validBuy1 := &supplied.LiquidityOrder{
		Price:      num.NewUint(minPriceInt),
		Proportion: 20,
	}
	validBuy2 := &supplied.LiquidityOrder{
		Price:      num.NewUint(minPriceInt + 1),
		Proportion: 30,
	}
	buyShapes := []*supplied.LiquidityOrder{
		validBuy1,
		validBuy2,
	}
	validSell1 := &supplied.LiquidityOrder{
		Price:      num.NewUint(maxPriceInt - 1),
		Proportion: 11,
	}
	validSell2 := &supplied.LiquidityOrder{
		Price:      num.NewUint(maxPriceInt),
		Proportion: 22,
	}
	sellShapes := []*supplied.LiquidityOrder{
		validSell1,
		validSell2,
	}
	validBuy1Prob := 0.1
	validBuy2Prob := 0.2
	validSell1Prob := 0.22
	validSell2Prob := 0.11
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, validBuy1.Price.Float64(), true, true, minPrice, maxPrice).Return(validBuy1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, validBuy2.Price.Float64(), true, true, minPrice, maxPrice).Return(validBuy2Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, validSell1.Price.Float64(), false, true, minPrice, maxPrice).Return(validSell1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, validSell2.Price.Float64(), false, true, minPrice, maxPrice).Return(validSell2Prob).Times(1)

	engine := supplied.NewEngine(riskModel, priceMonitor)
	require.NotNil(t, engine)

	// Negative liquidity obligation -> 0 sizes on all orders
	liquidityObligation := -2.5
	err := engine.CalculateLiquidityImpliedVolumes(MarkPrice, MarkPrice, liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	var zero uint64 = 0
	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity := engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, allOrders)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)

	// 0 liquidity obligation -> 0 sizes on all orders
	liquidityObligation = 0
	err = engine.CalculateLiquidityImpliedVolumes(MarkPrice, MarkPrice, liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, allOrders)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)

	// Positive liquidity obligation -> positive sizes on orders -> suplied liquidity >= liquidity obligation
	liquidityObligation = 25
	err = engine.CalculateLiquidityImpliedVolumes(MarkPrice, MarkPrice, liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	expectedVolumeValidBuy1 := uint64(math.Ceil(liquidityObligation * float64(validBuy1.Proportion) / float64(validBuy1.Proportion+validBuy2.Proportion) / validBuy1Prob / validBuy1.Price.Float64()))
	expectedVolumeValidBuy2 := uint64(math.Ceil(liquidityObligation * float64(validBuy2.Proportion) / float64(validBuy1.Proportion+validBuy2.Proportion) / validBuy2Prob / validBuy2.Price.Float64()))

	expectedVolumeValidSell1 := uint64(math.Ceil(liquidityObligation * float64(validSell1.Proportion) / float64(validSell1.Proportion+validSell2.Proportion) / validSell1Prob / validSell1.Price.Float64()))
	expectedVolumeValidSell2 := uint64(math.Ceil(liquidityObligation * float64(validSell2.Proportion) / float64(validSell1.Proportion+validSell2.Proportion) / validSell2Prob / validSell2.Price.Float64()))

	require.Equal(t, expectedVolumeValidBuy1, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, expectedVolumeValidBuy2, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, expectedVolumeValidSell1, validSell1.LiquidityImpliedVolume)
	require.Equal(t, expectedVolumeValidSell2, validSell2.LiquidityImpliedVolume)

	// Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, allOrders)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)
	require.True(t, totalSuppliedLiquidity < 2*liquidityObligation)

}

func TestCalculateLiquidityImpliedSizes_WithLimitOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minPrice := 89.2
	maxPrice := 111.1
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(12)

	var minPriceInt = uint64(math.Ceil(minPrice))
	var maxPriceInt = uint64(math.Floor(maxPrice))

	validBuy1 := &supplied.LiquidityOrder{
		Price:      num.NewUint(minPriceInt),
		Proportion: 20,
	}
	validBuy2 := &supplied.LiquidityOrder{
		Price:      num.NewUint(minPriceInt + 1),
		Proportion: 30,
	}
	buyShapes := []*supplied.LiquidityOrder{
		validBuy1,
		validBuy2,
	}
	validSell1 := &supplied.LiquidityOrder{
		Price:      num.NewUint(maxPriceInt - 1),
		Proportion: 11,
	}
	validSell2 := &supplied.LiquidityOrder{
		Price:      num.NewUint(maxPriceInt),
		Proportion: 22,
	}
	sellShapes := []*supplied.LiquidityOrder{
		validSell1,
		validSell2,
	}

	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, validBuy1.Price.Float64(), true, true, minPrice, maxPrice).Return(0.1).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, validBuy2.Price.Float64(), true, true, minPrice, maxPrice).Return(0.2).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, validSell1.Price.Float64(), false, true, minPrice, maxPrice).Return(0.22).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, validSell2.Price.Float64(), false, true, minPrice, maxPrice).Return(0.11).Times(1)

	engine := supplied.NewEngine(riskModel, priceMonitor)
	require.NotNil(t, engine)

	liquidityObligation := 123.45
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

	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, limitOrders[0].Price.Float64(), true, true, minPrice, maxPrice).Return(0.175).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, limitOrders[1].Price.Float64(), true, true, minPrice, maxPrice).Return(0.312).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, limitOrders[2].Price.Float64(), false, true, minPrice, maxPrice).Return(0.5).Times(1)

	limitOrdersSuppliedLiquidity := engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, collateOrders(limitOrders, nil, nil))
	require.True(t, limitOrdersSuppliedLiquidity < liquidityObligation)

	err := engine.CalculateLiquidityImpliedVolumes(MarkPrice, MarkPrice, liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	var zero uint64 = 0
	require.Less(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Less(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Less(t, zero, validSell1.LiquidityImpliedVolume)
	require.Less(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity := engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, allOrders)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)
	require.True(t, totalSuppliedLiquidity < 2*liquidityObligation)

	// Limit buy orders provide enoguh liquidity
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

	limitOrdersSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, collateOrders(limitOrders, nil, nil))
	require.True(t, limitOrdersSuppliedLiquidity < liquidityObligation)

	err = engine.CalculateLiquidityImpliedVolumes(MarkPrice, MarkPrice, liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Less(t, zero, validSell1.LiquidityImpliedVolume)
	require.Less(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, allOrders)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)
	require.True(t, totalSuppliedLiquidity < 2*liquidityObligation)

	//Limit sell orders provide enoguh liquidity
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

	limitOrdersSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, collateOrders(limitOrders, nil, nil))
	require.True(t, limitOrdersSuppliedLiquidity < liquidityObligation)

	err = engine.CalculateLiquidityImpliedVolumes(MarkPrice, MarkPrice, liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Less(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Less(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, allOrders)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)
	require.True(t, totalSuppliedLiquidity < 2*liquidityObligation)

	// Limit buy & sell orders provide enoguh liquidity
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

	limitOrdersSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, collateOrders(limitOrders, nil, nil))
	require.True(t, limitOrdersSuppliedLiquidity > liquidityObligation)

	err = engine.CalculateLiquidityImpliedVolumes(MarkPrice, MarkPrice, liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders = collateOrders(limitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity = engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, allOrders)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)
}

func TestCalculateLiquidityImpliedSizes_NoValidOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	priceMonitor := mocks.NewMockPriceMonitor(ctrl)
	riskModel.EXPECT().GetProjectionHorizon().Return(Horizon).Times(1)
	minPrice := 89.2
	maxPrice := 111.1
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(2)

	limitOrders := []*types.Order{}

	var minPriceInt = uint64(math.Ceil(minPrice))
	var maxPriceInt = uint64(math.Floor(maxPrice))

	invalidBuy := &supplied.LiquidityOrder{
		Price:      num.NewUint(minPriceInt - 1),
		Proportion: 10,
	}
	buyShapes := []*supplied.LiquidityOrder{
		invalidBuy,
	}
	invalidSell := &supplied.LiquidityOrder{
		Price:      num.NewUint(maxPriceInt + 1),
		Proportion: 33,
	}
	sellShapes := []*supplied.LiquidityOrder{
		invalidSell,
	}
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, invalidBuy.Price.Float64(), true, true, minPrice, maxPrice).Return(0.0).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, invalidSell.Price.Float64(), false, true, minPrice, maxPrice).Return(0.0).Times(1)

	engine := supplied.NewEngine(riskModel, priceMonitor)
	require.NotNil(t, engine)

	liquidityObligation := 20.0
	// Expecting no error now (other component assures orders get shifted to valid price range, failsafe in place to safeguard against near-zero probability of trading)
	err := engine.CalculateLiquidityImpliedVolumes(MarkPrice, MarkPrice, liquidityObligation, limitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	// We do expect an error when no orders specified though.
	err = engine.CalculateLiquidityImpliedVolumes(MarkPrice, MarkPrice, liquidityObligation, limitOrders, []*supplied.LiquidityOrder{}, []*supplied.LiquidityOrder{})
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
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, order1.Price.Float64(), true, true, minPrice, maxPrice).Return(0.123).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, order2.Price.Float64(), false, true, minPrice, maxPrice).Return(0.234).Times(1)

	engine := supplied.NewEngine(riskModel, priceMonitor)
	require.NotNil(t, engine)

	liquidity1 := engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, orders)
	require.Less(t, 0.0, liquidity1)

	// Change minPrice, maxPrice and verify that probability of trading is called with new values
	minPrice -= 10
	maxPrice += 10
	priceMonitor.EXPECT().GetValidPriceRange().Return(minPrice, maxPrice).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, order1.Price.Float64(), true, true, minPrice, maxPrice).Return(0.123).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(MarkPrice.Float64(), Horizon, order2.Price.Float64(), false, true, minPrice, maxPrice).Return(0.234).Times(1)

	liquidity2 := engine.CalculateSuppliedLiquidity(MarkPrice, MarkPrice, orders)
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
