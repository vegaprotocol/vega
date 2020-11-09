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
