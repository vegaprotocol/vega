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
	riskModel.EXPECT().ProbabilityOfTrading(float64(buyOrder1.Price), true, true, minPrice, maxPrice).Return(buyOrder1Prob).Times(1)
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
	riskModel.EXPECT().ProbabilityOfTrading(float64(buyOrder1.Price), true, true, minPrice, maxPrice).Return(buyOrder1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(sellOrder1.Price), false, true, minPrice, maxPrice).Return(sellOrder1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(sellOrder2.Price), false, true, minPrice, maxPrice).Return(sellOrder2Prob).Times(1)

	liquidity, err = engine.CalculateSuppliedLiquidity([]types.Order{buyOrder1, sellOrder1, sellOrder2, buyOrder2})
	require.NoError(t, err)
	require.Equal(t, expectedLiquidity, liquidity)
}

func TestCalculateLiquidityImpliedSizes_NoLimitOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
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
	riskModel.EXPECT().ProbabilityOfTrading(float64(invalidBuy.Price), true, true, minPrice, maxPrice).Return(0.0).Times(4)
	riskModel.EXPECT().ProbabilityOfTrading(float64(validBuy1.Price), true, true, minPrice, maxPrice).Return(validBuy1Prob).Times(4)
	riskModel.EXPECT().ProbabilityOfTrading(float64(validBuy2.Price), true, true, minPrice, maxPrice).Return(validBuy2Prob).Times(4)
	riskModel.EXPECT().ProbabilityOfTrading(float64(invalidSell.Price), false, true, minPrice, maxPrice).Return(0.0).Times(4)
	riskModel.EXPECT().ProbabilityOfTrading(float64(validSell1.Price), false, true, minPrice, maxPrice).Return(validSell1Prob).Times(4)
	riskModel.EXPECT().ProbabilityOfTrading(float64(validSell2.Price), false, true, minPrice, maxPrice).Return(validSell2Prob).Times(4)

	engine := supplied.NewEngine(riskModel, rangeProvider)
	require.NotNil(t, engine)

	// Negative liquidity obligation -> 0 sizes on all orders
	liquidityObligation := -2.5
	err := engine.CalculateLiquidityImpliedSizes(liquidityObligation, buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	var zero uint64 = 0
	require.Equal(t, zero, invalidBuy.LiquidityImpliedSize)
	require.Equal(t, zero, invalidSell.LiquidityImpliedSize)
	require.Equal(t, zero, validBuy1.LiquidityImpliedSize)
	require.Equal(t, zero, validBuy2.LiquidityImpliedSize)
	require.Equal(t, zero, validSell1.LiquidityImpliedSize)
	require.Equal(t, zero, validSell2.LiquidityImpliedSize)

	// 	Verify engine is internally consistent
	allOrders := collateOrders(buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity, err := engine.CalculateSuppliedLiquidity(allOrders)
	require.NoError(t, err)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)

	// 0 liquidity obligation -> 0 sizes on all orders
	liquidityObligation = 0
	err = engine.CalculateLiquidityImpliedSizes(liquidityObligation, buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, invalidBuy.LiquidityImpliedSize)
	require.Equal(t, zero, invalidSell.LiquidityImpliedSize)
	require.Equal(t, zero, validBuy1.LiquidityImpliedSize)
	require.Equal(t, zero, validBuy2.LiquidityImpliedSize)
	require.Equal(t, zero, validSell1.LiquidityImpliedSize)
	require.Equal(t, zero, validSell2.LiquidityImpliedSize)

	// Verify engine is internally consistent
	allOrders = collateOrders(buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity, err = engine.CalculateSuppliedLiquidity(allOrders)
	require.NoError(t, err)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)

	// Positive liquidity obligation -> positive sizes on orders -> suplied liquidity >= liquidity obligation
	liquidityObligation = 25
	err = engine.CalculateLiquidityImpliedSizes(liquidityObligation, buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	require.NoError(t, err)

	require.Equal(t, zero, invalidBuy.LiquidityImpliedSize)
	require.Equal(t, zero, invalidSell.LiquidityImpliedSize)

	expectedSizeValidBuy1 := uint64(math.Ceil(liquidityObligation * float64(validBuy1.Proportion) / float64((validBuy1.Proportion + validBuy2.Proportion)) / validBuy1Prob))
	expectedSizeValidBuy2 := uint64(math.Ceil(liquidityObligation * float64(validBuy2.Proportion) / float64((validBuy1.Proportion + validBuy2.Proportion)) / validBuy2Prob))

	expectedSizeValidSell1 := uint64(math.Ceil(liquidityObligation * float64(validSell1.Proportion) / float64((validSell1.Proportion + validSell2.Proportion)) / validSell1Prob))
	expectedSizeValidSell2 := uint64(math.Ceil(liquidityObligation * float64(validSell2.Proportion) / float64((validSell1.Proportion + validSell2.Proportion)) / validSell2Prob))

	require.Equal(t, expectedSizeValidBuy1, validBuy1.LiquidityImpliedSize)
	require.Equal(t, expectedSizeValidBuy2, validBuy2.LiquidityImpliedSize)
	require.Equal(t, expectedSizeValidSell1, validSell1.LiquidityImpliedSize)
	require.Equal(t, expectedSizeValidSell2, validSell2.LiquidityImpliedSize)

	// Verify engine is internally consistent
	allOrders = collateOrders(buyLimitOrders, sellLimitOrders, buyShapes, sellShapes)
	totalSuppliedLiquidity, err = engine.CalculateSuppliedLiquidity(allOrders)
	require.NoError(t, err)
	require.True(t, totalSuppliedLiquidity >= liquidityObligation)

}

func collateOrders(buyLimitOrders []types.Order, sellLimitOrders []types.Order, buyShapes []*supplied.LiquidityOrder, sellShapes []*supplied.LiquidityOrder) []types.Order {
	allOrders := make([]types.Order, 0, len(buyLimitOrders)+len(sellLimitOrders)+len(buyShapes)+len(sellShapes))

	allOrders = append(allOrders, buyLimitOrders...)
	allOrders = append(allOrders, sellLimitOrders...)

	for _, s := range buyShapes {
		lo := types.Order{
			Price:     s.Price,
			Size:      s.LiquidityImpliedSize,
			Remaining: s.LiquidityImpliedSize,
			Side:      types.Side_SIDE_BUY,
		}
		allOrders = append(allOrders, lo)
	}

	for _, s := range sellShapes {
		lo := types.Order{
			Price:     s.Price,
			Size:      s.LiquidityImpliedSize,
			Remaining: s.LiquidityImpliedSize,
			Side:      types.Side_SIDE_SELL,
		}
		allOrders = append(allOrders, lo)
	}

	return allOrders

}
