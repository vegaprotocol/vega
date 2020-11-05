package supplied_test

import (
	"testing"

	"code.vegaprotocol.io/vega/liquidity/supplied"
	"code.vegaprotocol.io/vega/liquidity/supplied/mocks"
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
