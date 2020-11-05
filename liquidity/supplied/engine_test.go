package supplied_test

import (
	"errors"
	"math"
	"testing"

	"code.vegaprotocol.io/vega/liquidity/supplied"
	"code.vegaprotocol.io/vega/liquidity/supplied/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

const MarketID string = "TestMarket"

func TestConstructor(t *testing.T) {
	ctrl := gomock.NewController(t)
	lpProvider := mocks.NewMockLiquidityProvisionProvider(ctrl)
	orderProvider := mocks.NewMockOrderProvider(ctrl)
	riskModel := mocks.NewMockRiskModel(ctrl)

	engine, err := supplied.NewEngine(MarketID, lpProvider, orderProvider, riskModel)
	require.NotNil(t, engine)
	require.NoError(t, err)

	engine, err = supplied.NewEngine(MarketID, nil, orderProvider, riskModel)
	require.Nil(t, engine)
	require.Error(t, err)

	engine, err = supplied.NewEngine(MarketID, lpProvider, nil, riskModel)
	require.Nil(t, engine)
	require.Error(t, err)

	engine, err = supplied.NewEngine(MarketID, lpProvider, orderProvider, nil)
	require.Nil(t, engine)
	require.Error(t, err)
}

func TestGetSuppliedLiquidity(t *testing.T) {
	ctrl := gomock.NewController(t)
	lpProvider := mocks.NewMockLiquidityProvisionProvider(ctrl)
	orderProvider := mocks.NewMockOrderProvider(ctrl)
	riskModel := mocks.NewMockRiskModel(ctrl)

	minPrice := 89.2
	maxPrice := 111.1

	// LP provider error
	errString := "liquidity provider error"

	lps := []types.LiquidityProvision{}
	lpProvider.EXPECT().GetLiquidityProvisions(MarketID).Return(nil, errors.New(errString)).Times(1)

	engine, err := supplied.NewEngine(MarketID, lpProvider, orderProvider, riskModel)
	require.NotNil(t, engine)
	require.NoError(t, err)

	liquidity, err := engine.GetSuppliedLiquidity()
	require.Error(t, err)
	require.EqualError(t, err, errString)
	require.Equal(t, 0.0, liquidity)

	// No LP orders
	lps = []types.LiquidityProvision{}
	riskModel.EXPECT().PriceRange().Return(minPrice, maxPrice).Times(1)
	lpProvider.EXPECT().GetLiquidityProvisions(MarketID).Return(lps, nil).Times(1)

	engine, err = supplied.NewEngine(MarketID, lpProvider, orderProvider, riskModel)
	require.NotNil(t, engine)
	require.NoError(t, err)

	liquidity, err = engine.GetSuppliedLiquidity()
	require.NoError(t, err)
	require.Equal(t, 0.0, liquidity)

	// 1 LP order, no buys, no sells
	lp1 := types.LiquidityProvision{
		Buys:  []*types.LiquidityOrderReference{},
		Sells: []*types.LiquidityOrderReference{},
	}
	lps = []types.LiquidityProvision{lp1}
	riskModel.EXPECT().PriceRange().Return(minPrice, maxPrice).Times(1)
	lpProvider.EXPECT().GetLiquidityProvisions(MarketID).Return(lps, nil).Times(1)

	engine, err = supplied.NewEngine(MarketID, lpProvider, orderProvider, riskModel)
	require.NotNil(t, engine)
	require.NoError(t, err)

	liquidity, err = engine.GetSuppliedLiquidity()
	require.NoError(t, err)
	require.Equal(t, 0.0, liquidity)

	// 1 LP order, order provider error
	errString = "order provider error"
	lp1buyOrder1Ref := "lp1buyOrder1"
	lp1buyOrder1 := &types.Order{
		Id:        lp1buyOrder1Ref,
		Price:     102,
		Size:      30,
		Remaining: 25,
	}
	lp1 = types.LiquidityProvision{
		Buys: []*types.LiquidityOrderReference{
			{
				OrderID: lp1buyOrder1Ref,
			},
		},
		Sells: []*types.LiquidityOrderReference{},
	}
	lps = []types.LiquidityProvision{lp1}
	lp1buyOrder1Prob := 0.256

	riskModel.EXPECT().PriceRange().Return(minPrice, maxPrice).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(lp1buyOrder1.Price), true, true, minPrice, maxPrice).Return(lp1buyOrder1Prob).Times(1)
	lpProvider.EXPECT().GetLiquidityProvisions(MarketID).Return(lps, nil).Times(1)
	orderProvider.EXPECT().GetOrderByID(lp1buyOrder1Ref).Return(nil, errors.New(errString)).Times(1)

	engine, err = supplied.NewEngine(MarketID, lpProvider, orderProvider, riskModel)
	require.NotNil(t, engine)
	require.NoError(t, err)

	liquidity, err = engine.GetSuppliedLiquidity()
	require.Error(t, err)
	require.EqualError(t, err, errString)
	require.Equal(t, 0.0, liquidity)

	// 1 LP order, 1 buy, no sells
	riskModel.EXPECT().PriceRange().Return(minPrice, maxPrice).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(lp1buyOrder1.Price), true, true, minPrice, maxPrice).Return(lp1buyOrder1Prob).Times(1)
	lpProvider.EXPECT().GetLiquidityProvisions(MarketID).Return(lps, nil).Times(1)
	orderProvider.EXPECT().GetOrderByID(lp1buyOrder1Ref).Return(lp1buyOrder1, nil).Times(1)

	engine, err = supplied.NewEngine(MarketID, lpProvider, orderProvider, riskModel)
	require.NotNil(t, engine)
	require.NoError(t, err)

	liquidity, err = engine.GetSuppliedLiquidity()
	require.NoError(t, err)
	require.Equal(t, 0.0, liquidity)

	// 1 LP order, 1 buy, 2 sells
	lp1sellOrder1Ref := "lp1sellOrder1"
	lp1sellOrder2Ref := "lp1sellOrder2"
	lp1sellOrder1 := &types.Order{
		Id:        lp1sellOrder1Ref,
		Price:     99,
		Size:      15,
		Remaining: 11,
	}
	lp1sellOrder2 := &types.Order{
		Id:        lp1sellOrder1Ref,
		Price:     97,
		Size:      60,
		Remaining: 60,
	}

	lp1 = types.LiquidityProvision{
		Buys: []*types.LiquidityOrderReference{
			{
				OrderID: lp1buyOrder1Ref,
			},
		},
		Sells: []*types.LiquidityOrderReference{
			{
				OrderID: lp1sellOrder1Ref,
			},
			{
				OrderID: lp1sellOrder2Ref,
			},
		},
	}
	lps = []types.LiquidityProvision{lp1}
	lp1sellOrder1Prob := 0.33
	lp1sellOrder2Prob := 0.17
	lp1buyLiquidity := float64(lp1buyOrder1.Price) * float64(lp1buyOrder1.Remaining) * lp1buyOrder1Prob
	lp1sellLiquidity := float64(lp1sellOrder1.Price)*float64(lp1sellOrder1.Remaining)*lp1sellOrder1Prob + float64(lp1sellOrder2.Price)*float64(lp1sellOrder2.Remaining)*lp1sellOrder2Prob
	expectedLiquidity := math.Min(lp1buyLiquidity, lp1sellLiquidity)

	riskModel.EXPECT().PriceRange().Return(minPrice, maxPrice)
	riskModel.EXPECT().ProbabilityOfTrading(float64(lp1buyOrder1.Price), true, true, minPrice, maxPrice).Return(lp1buyOrder1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(lp1sellOrder1.Price), false, true, minPrice, maxPrice).Return(lp1sellOrder1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(lp1sellOrder2.Price), false, true, minPrice, maxPrice).Return(lp1sellOrder2Prob).Times(1)
	lpProvider.EXPECT().GetLiquidityProvisions(MarketID).Return(lps, nil).Times(1)
	orderProvider.EXPECT().GetOrderByID(lp1buyOrder1Ref).Return(lp1buyOrder1, nil).Times(1)
	orderProvider.EXPECT().GetOrderByID(lp1sellOrder1Ref).Return(lp1sellOrder1, nil).Times(1)
	orderProvider.EXPECT().GetOrderByID(lp1sellOrder2Ref).Return(lp1sellOrder2, nil).Times(1)

	engine, err = supplied.NewEngine(MarketID, lpProvider, orderProvider, riskModel)
	require.NotNil(t, engine)
	require.NoError(t, err)

	liquidity, err = engine.GetSuppliedLiquidity()
	require.NoError(t, err)
	require.Equal(t, expectedLiquidity, liquidity)

	// 2 LP orders
	lp2buyOrder1Ref := "lp2buyOrder1"
	lp2buyOrder1 := &types.Order{
		Id:        lp1buyOrder1Ref,
		Price:     102,
		Size:      600,
		Remaining: 599,
	}
	lp2 := types.LiquidityProvision{
		Buys: []*types.LiquidityOrderReference{
			{
				OrderID: lp2buyOrder1Ref,
			},
		},
		Sells: []*types.LiquidityOrderReference{},
	}
	lps = []types.LiquidityProvision{lp1, lp2}

	lp2buyLiquidity := float64(lp2buyOrder1.Price) * float64(lp2buyOrder1.Remaining) * lp1buyOrder1Prob
	expectedLiquidity = math.Min(lp1buyLiquidity+lp2buyLiquidity, lp1sellLiquidity)

	riskModel.EXPECT().PriceRange().Return(minPrice, maxPrice)
	riskModel.EXPECT().ProbabilityOfTrading(float64(lp1buyOrder1.Price), true, true, minPrice, maxPrice).Return(lp1buyOrder1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(lp1sellOrder1.Price), false, true, minPrice, maxPrice).Return(lp1sellOrder1Prob).Times(1)
	riskModel.EXPECT().ProbabilityOfTrading(float64(lp1sellOrder2.Price), false, true, minPrice, maxPrice).Return(lp1sellOrder2Prob).Times(1)
	lpProvider.EXPECT().GetLiquidityProvisions(MarketID).Return(lps, nil).Times(1)
	orderProvider.EXPECT().GetOrderByID(lp1buyOrder1Ref).Return(lp1buyOrder1, nil).Times(1)
	orderProvider.EXPECT().GetOrderByID(lp2buyOrder1Ref).Return(lp2buyOrder1, nil).Times(1)
	orderProvider.EXPECT().GetOrderByID(lp1sellOrder1Ref).Return(lp1sellOrder1, nil).Times(1)
	orderProvider.EXPECT().GetOrderByID(lp1sellOrder2Ref).Return(lp1sellOrder2, nil).Times(1)

	engine, err = supplied.NewEngine(MarketID, lpProvider, orderProvider, riskModel)
	require.NotNil(t, engine)
	require.NoError(t, err)

	liquidity, err = engine.GetSuppliedLiquidity()
	require.NoError(t, err)
	require.Equal(t, expectedLiquidity, liquidity)
}
