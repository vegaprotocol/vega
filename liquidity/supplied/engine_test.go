package supplied_test

import (
	"math"
	"testing"

	"code.vegaprotocol.io/vega/liquidity/supplied"
	"code.vegaprotocol.io/vega/liquidity/supplied/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCalculateSuppliedLiquidity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	rangeProvider := mocks.NewMockValidPriceRangeProvider(ctrl)

	minPrice := 89.2
	maxPrice := 111.1

	// No orders
	rangeProvider.EXPECT().ValidPriceRange().Return(minPrice, maxPrice).Times(1)

	engine := supplied.NewEngine(riskModel, rangeProvider)
	require.NotNil(t, engine)

	liquidity, err := engine.CalculateSuppliedLiquidity([]types.Order{})
	require.NoError(t, err)
	require.Equal(t, 0.0, liquidity)

	// 1 buy, no sells
	buyOrder1 := types.Order{
		Price:     102,
		Size:      30,
		Remaining: 25,
		Side:      types.Side_SIDE_BUY,
	}

	buyOrder1Prob := 0.256
	rangeProvider.EXPECT().ValidPriceRange().Return(minPrice, maxPrice).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(buyOrder1.Price), true, true, minPrice, maxPrice).Return(buyOrder1Prob).Times(1)

	liquidity, err = engine.CalculateSuppliedLiquidity([]types.Order{buyOrder1})
	require.NoError(t, err)
	require.Equal(t, 0.0, liquidity)

	// 1 buy, 2 sells
	sellOrder1 := types.Order{
		Price:     99,
		Size:      15,
		Remaining: 11,
		Side:      types.Side_SIDE_SELL,
	}
	sellOrder2 := types.Order{
		Price:     97,
		Size:      60,
		Remaining: 60,
		Side:      types.Side_SIDE_SELL,
	}

	sellOrder1Prob := 0.33
	sellOrder2Prob := 0.17
	buyLiquidity := float64(buyOrder1.Price) * float64(buyOrder1.Remaining) * buyOrder1Prob
	sellLiquidity := float64(sellOrder1.Price)*float64(sellOrder1.Remaining)*sellOrder1Prob + float64(sellOrder2.Price)*float64(sellOrder2.Remaining)*sellOrder2Prob
	expectedLiquidity := math.Min(buyLiquidity, sellLiquidity)

	rangeProvider.EXPECT().ValidPriceRange().Return(minPrice, maxPrice)
	riskModel.EXPECT().ProbabilityOfTrading(float64(sellOrder1.Price), false, true, minPrice, maxPrice).Return(sellOrder1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(sellOrder2.Price), false, true, minPrice, maxPrice).Return(sellOrder2Prob).Times(1)

	liquidity, err = engine.CalculateSuppliedLiquidity([]types.Order{buyOrder1, sellOrder1, sellOrder2})
	require.NoError(t, err)
	require.Equal(t, expectedLiquidity, liquidity)

	// 2 buys, 2 sells
	buyOrder2 := types.Order{
		Price:     102,
		Size:      600,
		Remaining: 599,
		Side:      types.Side_SIDE_BUY,
	}

	buyLiquidity += float64(buyOrder2.Price) * float64(buyOrder2.Remaining) * buyOrder1Prob
	expectedLiquidity = math.Min(buyLiquidity, sellLiquidity)

	rangeProvider.EXPECT().ValidPriceRange().Return(minPrice, maxPrice)

	liquidity, err = engine.CalculateSuppliedLiquidity([]types.Order{buyOrder1, sellOrder1, sellOrder2, buyOrder2})
	require.NoError(t, err)
	require.Equal(t, expectedLiquidity, liquidity)
}

func Test_InteralConsistency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	rangeProvider := mocks.NewMockValidPriceRangeProvider(ctrl)
	minPrice := 89.2
	maxPrice := 111.1
	rangeProvider.EXPECT().ValidPriceRange().Return(minPrice, maxPrice).Times(2)

	buyLimitOrders := []types.Order{}
	sellLimitOrders := []types.Order{}

	var minPriceInt uint64 = uint64(math.Ceil(minPrice))
	var maxPriceInt uint64 = uint64(math.Floor(maxPrice))

	buy := &supplied.LiquidityOrder{
		Price:      minPriceInt,
		Proportion: 1,
	}
	buyShapes := []*supplied.LiquidityOrder{
		buy,
	}

	sell := &supplied.LiquidityOrder{
		Price:      maxPriceInt,
		Proportion: 1,
	}

	sellShapes := []*supplied.LiquidityOrder{
		sell,
	}
	validBuy1Prob := 0.1
	validSell1Prob := 0.22
	riskModel.EXPECT().ProbabilityOfTrading(float64(buy.Price), true, true, minPrice, maxPrice).Return(validBuy1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(sell.Price), false, true, minPrice, maxPrice).Return(validSell1Prob).Times(1)

	engine := supplied.NewEngine(riskModel, rangeProvider)
	require.NotNil(t, engine)

	// Negative liquidity obligation -> 0 sizes on all orders
	liquidityObligation := 100.0
	err := engine.CalculateLiquidityImpliedVolumes(liquidityObligation, buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	var zero uint64 = 0
	require.Less(t, zero, buy.LiquidityImpliedVolume)
	require.Less(t, zero, sell.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity, err := engine.CalculateSuppliedLiquidity(allOrders)
	require.NoError(t, err)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)
}

func TestCalculateLiquidityImpliedSizes_NoLimitOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	rangeProvider := mocks.NewMockValidPriceRangeProvider(ctrl)
	minPrice := 89.2
	maxPrice := 111.1
	rangeProvider.EXPECT().ValidPriceRange().Return(minPrice, maxPrice).Times(6)

	buyLimitOrders := []types.Order{}
	sellLimitOrders := []types.Order{}

	var minPriceInt uint64 = uint64(math.Ceil(minPrice))
	var maxPriceInt uint64 = uint64(math.Floor(maxPrice))

	invalidBuy := &supplied.LiquidityOrder{
		Price:      minPriceInt - 1,
		Proportion: 10,
	}
	validBuy1 := &supplied.LiquidityOrder{
		Price:      minPriceInt,
		Proportion: 20,
	}
	validBuy2 := &supplied.LiquidityOrder{
		Price:      minPriceInt + 1,
		Proportion: 30,
	}
	buyShapes := []*supplied.LiquidityOrder{
		invalidBuy,
		validBuy1,
		validBuy2,
	}
	validSell1 := &supplied.LiquidityOrder{
		Price:      maxPriceInt - 1,
		Proportion: 11,
	}
	validSell2 := &supplied.LiquidityOrder{
		Price:      maxPriceInt,
		Proportion: 22,
	}
	invalidSell := &supplied.LiquidityOrder{
		Price:      maxPriceInt + 1,
		Proportion: 33,
	}
	sellShapes := []*supplied.LiquidityOrder{
		validSell1,
		validSell2,
		invalidSell,
	}
	validBuy1Prob := 0.1
	validBuy2Prob := 0.2
	validSell1Prob := 0.22
	validSell2Prob := 0.11
	riskModel.EXPECT().ProbabilityOfTrading(float64(invalidBuy.Price), true, true, minPrice, maxPrice).Return(0.0).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(validBuy1.Price), true, true, minPrice, maxPrice).Return(validBuy1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(validBuy2.Price), true, true, minPrice, maxPrice).Return(validBuy2Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(invalidSell.Price), false, true, minPrice, maxPrice).Return(0.0).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(validSell1.Price), false, true, minPrice, maxPrice).Return(validSell1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(validSell2.Price), false, true, minPrice, maxPrice).Return(validSell2Prob).Times(1)

	engine := supplied.NewEngine(riskModel, rangeProvider)
	require.NotNil(t, engine)

	// Negative liquidity obligation -> 0 sizes on all orders
	liquidityObligation := -2.5
	err := engine.CalculateLiquidityImpliedVolumes(liquidityObligation, buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	var zero uint64 = 0
	require.Equal(t, zero, invalidBuy.LiquidityImpliedVolume)
	require.Equal(t, zero, invalidSell.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity, err := engine.CalculateSuppliedLiquidity(allOrders)
	require.NoError(t, err)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)

	// 0 liquidity obligation -> 0 sizes on all orders
	liquidityObligation = 0
	err = engine.CalculateLiquidityImpliedVolumes(liquidityObligation, buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, invalidBuy.LiquidityImpliedVolume)
	require.Equal(t, zero, invalidSell.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// Verify engine is internally consistent
	allOrders = collateOrders(buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity, err = engine.CalculateSuppliedLiquidity(allOrders)
	require.NoError(t, err)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)

	// Positive liquidity obligation -> positive sizes on orders -> suplied liquidity >= liquidity obligation
	liquidityObligation = 25
	err = engine.CalculateLiquidityImpliedVolumes(liquidityObligation, buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, invalidBuy.LiquidityImpliedVolume)
	require.Equal(t, zero, invalidSell.LiquidityImpliedVolume)

	expectedVolumeValidBuy1 := uint64(math.Ceil(liquidityObligation * float64(validBuy1.Proportion) / float64((validBuy1.Proportion + validBuy2.Proportion)) / validBuy1Prob / float64(validBuy1.Price)))
	expectedVolumeValidBuy2 := uint64(math.Ceil(liquidityObligation * float64(validBuy2.Proportion) / float64((validBuy1.Proportion + validBuy2.Proportion)) / validBuy2Prob / float64(validBuy2.Price)))

	expectedVolumeValidSell1 := uint64(math.Ceil(liquidityObligation * float64(validSell1.Proportion) / float64((validSell1.Proportion + validSell2.Proportion)) / validSell1Prob / float64(validSell1.Price)))
	expectedVolumeValidSell2 := uint64(math.Ceil(liquidityObligation * float64(validSell2.Proportion) / float64((validSell1.Proportion + validSell2.Proportion)) / validSell2Prob / float64(validSell2.Price)))

	require.Equal(t, expectedVolumeValidBuy1, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, expectedVolumeValidBuy2, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, expectedVolumeValidSell1, validSell1.LiquidityImpliedVolume)
	require.Equal(t, expectedVolumeValidSell2, validSell2.LiquidityImpliedVolume)

	// Verify engine is internally consistent
	allOrders = collateOrders(buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity, err = engine.CalculateSuppliedLiquidity(allOrders)
	require.NoError(t, err)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)
	require.True(t, totalSuppliedLiquidity < 2*liquidityObligation)

}

func TestCalculateLiquidityImpliedSizes_WithLimitOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	rangeProvider := mocks.NewMockValidPriceRangeProvider(ctrl)
	minPrice := 89.2
	maxPrice := 111.1
	rangeProvider.EXPECT().ValidPriceRange().Return(minPrice, maxPrice).Times(12)

	var minPriceInt uint64 = uint64(math.Ceil(minPrice))
	var maxPriceInt uint64 = uint64(math.Floor(maxPrice))

	invalidBuy := &supplied.LiquidityOrder{
		Price:      minPriceInt - 1,
		Proportion: 10,
	}
	validBuy1 := &supplied.LiquidityOrder{
		Price:      minPriceInt,
		Proportion: 20,
	}
	validBuy2 := &supplied.LiquidityOrder{
		Price:      minPriceInt + 1,
		Proportion: 30,
	}
	buyShapes := []*supplied.LiquidityOrder{
		invalidBuy,
		validBuy1,
		validBuy2,
	}
	validSell1 := &supplied.LiquidityOrder{
		Price:      maxPriceInt - 1,
		Proportion: 11,
	}
	validSell2 := &supplied.LiquidityOrder{
		Price:      maxPriceInt,
		Proportion: 22,
	}
	invalidSell := &supplied.LiquidityOrder{
		Price:      maxPriceInt + 1,
		Proportion: 33,
	}
	sellShapes := []*supplied.LiquidityOrder{
		validSell1,
		validSell2,
		invalidSell,
	}

	riskModel.EXPECT().ProbabilityOfTrading(float64(invalidBuy.Price), true, true, minPrice, maxPrice).Return(0.0).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(validBuy1.Price), true, true, minPrice, maxPrice).Return(0.1).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(validBuy2.Price), true, true, minPrice, maxPrice).Return(0.2).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(invalidSell.Price), false, true, minPrice, maxPrice).Return(0.0).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(validSell1.Price), false, true, minPrice, maxPrice).Return(0.22).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(validSell2.Price), false, true, minPrice, maxPrice).Return(0.11).Times(1)

	engine := supplied.NewEngine(riskModel, rangeProvider)
	require.NotNil(t, engine)

	liquidityObligation := 123.45
	// Limit orders don't provide enough liquidity
	buyLimitOrders := []types.Order{
		{
			Price:     95,
			Size:      500,
			Remaining: 1,
			Side:      types.Side_SIDE_BUY,
		},
		{
			Price:     97,
			Size:      1000,
			Remaining: 1,
			Side:      types.Side_SIDE_BUY,
		},
	}
	sellLimitOrders := []types.Order{
		{
			Price:     104,
			Size:      500,
			Remaining: 1,
			Side:      types.Side_SIDE_SELL,
		},
	}

	riskModel.EXPECT().ProbabilityOfTrading(float64(buyLimitOrders[0].Price), true, true, minPrice, maxPrice).Return(0.175).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(buyLimitOrders[1].Price), true, true, minPrice, maxPrice).Return(0.312).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(sellLimitOrders[0].Price), false, true, minPrice, maxPrice).Return(0.5).Times(1)

	limitOrdersSuppliedLiquidity, err := engine.CalculateSuppliedLiquidity(collateOrders(buyLimitOrders, sellLimitOrders, nil, nil))
	require.NoError(t, err)
	require.True(t, limitOrdersSuppliedLiquidity < liquidityObligation)

	err = engine.CalculateLiquidityImpliedVolumes(liquidityObligation, buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	var zero uint64 = 0
	require.Equal(t, zero, invalidBuy.LiquidityImpliedVolume)
	require.Equal(t, zero, invalidSell.LiquidityImpliedVolume)
	require.Less(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Less(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Less(t, zero, validSell1.LiquidityImpliedVolume)
	require.Less(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity, err := engine.CalculateSuppliedLiquidity(allOrders)
	require.NoError(t, err)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)
	require.True(t, totalSuppliedLiquidity < 2*liquidityObligation)

	// Limit buy orders provide enoguh liquidity
	buyLimitOrders = []types.Order{
		{
			Price:     95,
			Size:      500,
			Remaining: 100,
			Side:      types.Side_SIDE_BUY,
		},
		{
			Price:     97,
			Size:      1000,
			Remaining: 100,
			Side:      types.Side_SIDE_BUY,
		},
	}
	sellLimitOrders = []types.Order{
		{
			Price:     104,
			Size:      500,
			Remaining: 1,
			Side:      types.Side_SIDE_SELL,
		},
	}

	limitOrdersSuppliedLiquidity, err = engine.CalculateSuppliedLiquidity(collateOrders(buyLimitOrders, sellLimitOrders, nil, nil))
	require.NoError(t, err)
	require.True(t, limitOrdersSuppliedLiquidity < liquidityObligation)

	err = engine.CalculateLiquidityImpliedVolumes(liquidityObligation, buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, invalidBuy.LiquidityImpliedVolume)
	require.Equal(t, zero, invalidSell.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Less(t, zero, validSell1.LiquidityImpliedVolume)
	require.Less(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders = collateOrders(buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity, err = engine.CalculateSuppliedLiquidity(allOrders)
	require.NoError(t, err)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)
	require.True(t, totalSuppliedLiquidity < 2*liquidityObligation)

	//Limit sell orders provide enoguh liquidity
	buyLimitOrders = []types.Order{
		{
			Price:     95,
			Size:      500,
			Remaining: 1,
			Side:      types.Side_SIDE_BUY,
		},
		{
			Price:     97,
			Size:      1000,
			Remaining: 1,
			Side:      types.Side_SIDE_BUY,
		},
	}
	sellLimitOrders = []types.Order{
		{
			Price:     104,
			Size:      500,
			Remaining: 100,
			Side:      types.Side_SIDE_SELL,
		},
	}

	limitOrdersSuppliedLiquidity, err = engine.CalculateSuppliedLiquidity(collateOrders(buyLimitOrders, sellLimitOrders, nil, nil))
	require.NoError(t, err)
	require.True(t, limitOrdersSuppliedLiquidity < liquidityObligation)

	err = engine.CalculateLiquidityImpliedVolumes(liquidityObligation, buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, invalidBuy.LiquidityImpliedVolume)
	require.Equal(t, zero, invalidSell.LiquidityImpliedVolume)
	require.Less(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Less(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders = collateOrders(buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity, err = engine.CalculateSuppliedLiquidity(allOrders)
	require.NoError(t, err)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)
	require.True(t, totalSuppliedLiquidity < 2*liquidityObligation)

	// Limit buy & sell orders provide enoguh liquidity
	buyLimitOrders = []types.Order{
		{
			Price:     95,
			Size:      500,
			Remaining: 100,
			Side:      types.Side_SIDE_BUY,
		},
		{
			Price:     97,
			Size:      1000,
			Remaining: 100,
			Side:      types.Side_SIDE_BUY,
		},
	}
	sellLimitOrders = []types.Order{
		{
			Price:     104,
			Size:      500,
			Remaining: 100,
			Side:      types.Side_SIDE_SELL,
		},
	}

	limitOrdersSuppliedLiquidity, err = engine.CalculateSuppliedLiquidity(collateOrders(buyLimitOrders, sellLimitOrders, nil, nil))
	require.NoError(t, err)
	require.True(t, limitOrdersSuppliedLiquidity > liquidityObligation)

	err = engine.CalculateLiquidityImpliedVolumes(liquidityObligation, buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, invalidBuy.LiquidityImpliedVolume)
	require.Equal(t, zero, invalidSell.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy1.LiquidityImpliedVolume)
	require.Equal(t, zero, validBuy2.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell1.LiquidityImpliedVolume)
	require.Equal(t, zero, validSell2.LiquidityImpliedVolume)

	// 	Verify engine is internally consistent
	allOrders = collateOrders(buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity, err = engine.CalculateSuppliedLiquidity(allOrders)
	require.NoError(t, err)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)
}

func TestCalculateLiquidityImpliedSizes_NoValidOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	rangeProvider := mocks.NewMockValidPriceRangeProvider(ctrl)
	minPrice := 89.2
	maxPrice := 111.1
	rangeProvider.EXPECT().ValidPriceRange().Return(minPrice, maxPrice).Times(2)

	buyLimitOrders := []types.Order{}
	sellLimitOrders := []types.Order{}

	var minPriceInt uint64 = uint64(math.Ceil(minPrice))
	var maxPriceInt uint64 = uint64(math.Floor(maxPrice))

	invalidBuy := &supplied.LiquidityOrder{
		Price:      minPriceInt - 1,
		Proportion: 10,
	}
	buyShapes := []*supplied.LiquidityOrder{
		invalidBuy,
	}
	invalidSell := &supplied.LiquidityOrder{
		Price:      maxPriceInt + 1,
		Proportion: 33,
	}
	sellShapes := []*supplied.LiquidityOrder{
		invalidSell,
	}
	riskModel.EXPECT().ProbabilityOfTrading(float64(invalidBuy.Price), true, true, minPrice, maxPrice).Return(0.0).Times(1)

	engine := supplied.NewEngine(riskModel, rangeProvider)
	require.NotNil(t, engine)

	liquidityObligation := 20.0
	// Expecting an error as limit orders don't cover the obligation there are no valid orders to calculate size for
	err := engine.CalculateLiquidityImpliedVolumes(liquidityObligation, buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	require.Error(t, err)

	// Same when there just aren't any buy/sell shapes
	err = engine.CalculateLiquidityImpliedVolumes(liquidityObligation, buyLimitOrders, sellLimitOrders, []*supplied.LiquidityOrder{}, []*supplied.LiquidityOrder{})
	require.Error(t, err)
}

func TestProbabilityOfTradingRecomputedAfterPriceRangeChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	riskModel := mocks.NewMockRiskModel(ctrl)
	rangeProvider := mocks.NewMockValidPriceRangeProvider(ctrl)
	minPrice := 89.2
	maxPrice := 111.1

	var minPriceInt uint64 = uint64(math.Ceil(minPrice))
	var maxPriceInt uint64 = uint64(math.Floor(maxPrice))

	order1 := types.Order{
		Price:     minPriceInt,
		Size:      15,
		Remaining: 11,
		Side:      types.Side_SIDE_BUY,
	}
	order2 := types.Order{
		Price:     maxPriceInt,
		Size:      60,
		Remaining: 60,
		Side:      types.Side_SIDE_SELL,
	}

	orders := []types.Order{
		order1,
		order2,
	}

	rangeProvider.EXPECT().ValidPriceRange().Return(minPrice, maxPrice).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(order1.Price), true, true, minPrice, maxPrice).Return(0.123).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(order2.Price), false, true, minPrice, maxPrice).Return(0.234).Times(1)

	engine := supplied.NewEngine(riskModel, rangeProvider)
	require.NotNil(t, engine)

	liquidity1, err := engine.CalculateSuppliedLiquidity(orders)
	require.NoError(t, err)
	require.Less(t, 0.0, liquidity1)

	// Change minPrice, maxPrice and verify that probability of trading is called with new values
	minPrice -= 10
	maxPrice += 10
	rangeProvider.EXPECT().ValidPriceRange().Return(minPrice, maxPrice).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(order1.Price), true, true, minPrice, maxPrice).Return(0.123).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(order2.Price), false, true, minPrice, maxPrice).Return(0.234).Times(1)

	liquidity2, err := engine.CalculateSuppliedLiquidity(orders)
	require.NoError(t, err)
	require.Less(t, 0.0, liquidity2)
	require.Equal(t, liquidity1, liquidity2)

}

func collateOrders(buyLimitOrders []types.Order, sellLimitOrders []types.Order, buyShapes []*supplied.LiquidityOrder, sellShapes []*supplied.LiquidityOrder) []types.Order {
	allOrders := make([]types.Order, 0, len(buyLimitOrders)+len(sellLimitOrders)+len(buyShapes)+len(sellShapes))

	allOrders = append(allOrders, buyLimitOrders...)
	allOrders = append(allOrders, sellLimitOrders...)

	for _, s := range buyShapes {
		lo := types.Order{
			Price:     s.Price,
			Size:      s.LiquidityImpliedVolume,
			Remaining: s.LiquidityImpliedVolume,
			Side:      types.Side_SIDE_BUY,
		}
		allOrders = append(allOrders, lo)
	}

	for _, s := range sellShapes {
		lo := types.Order{
			Price:     s.Price,
			Size:      s.LiquidityImpliedVolume,
			Remaining: s.LiquidityImpliedVolume,
			Side:      types.Side_SIDE_SELL,
		}
		allOrders = append(allOrders, lo)
	}
	return allOrders
}
