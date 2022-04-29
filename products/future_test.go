package products_test

import (
	"context"
	"testing"

	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/products"
	"code.vegaprotocol.io/vega/products/mocks"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestFuture(t *testing.T) {
	t.Run("Unsubscribing the oracle engine succeeds", testUnsubscribingTheOracleEngineSucceeds)
}

func TestScalingOfSettlementPrice(t *testing.T) {
	t.Run("No scaling needed for settlement price for asset decimals", testNoScalingNeeded)
	t.Run("Need to scale up the settlement price for asset decimals", testScalingUpNeeded)
	t.Run("Need to scale down the settlement price for asset decimals no loss of precision", testScalingDownNeeded)
	t.Run("Need to scale down the settlement price for asset decimals with loss of precision", testScalingDownNeededWithPrecisionLoss)
}

func testNoScalingNeeded(t *testing.T) {
	ft := testFuture(t)

	// settlement price is in 5 decimal places, asset in 5 decimal places => no scaling
	scaled, err := ft.future.ScaleSettlementPriceToDecimalPlaces(num.NewUint(100000), 5)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(100000), scaled)
}

func testScalingUpNeeded(t *testing.T) {
	ft := testFuture(t)

	// settlement price is in 5 decimal places, asset in 10 decimal places => x10^5
	scaled, err := ft.future.ScaleSettlementPriceToDecimalPlaces(num.NewUint(100000), 10)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(10000000000), scaled)
}

func testScalingDownNeeded(t *testing.T) {
	ft := testFuture(t)

	// settlement price is in 5 decimal places, asset in 3 decimal places => x10^-2
	scaled, err := ft.future.ScaleSettlementPriceToDecimalPlaces(num.NewUint(100000), 3)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(1000), scaled)
}

func testScalingDownNeededWithPrecisionLoss(t *testing.T) {
	ft := testFuture(t)

	// settlement price is in 5 decimal places, asset in 3 decimal places => x10^-2
	scaled, err := ft.future.ScaleSettlementPriceToDecimalPlaces(num.NewUint(123456), 3)
	require.NoError(t, err)
	require.Equal(t, num.NewUint(1234), scaled)
}

func testUnsubscribingTheOracleEngineSucceeds(t *testing.T) {
	// given
	tf := testFuture(t)
	ctx := context.Background()

	// expect
	tf.oe.EXPECT().Unsubscribe(ctx, oracles.SubscriptionID(1))
	tf.oe.EXPECT().Unsubscribe(ctx, oracles.SubscriptionID(2))

	// when
	tf.future.Unsubscribe(context.Background(), tf.oe)
}

type tstFuture struct {
	oe     *mocks.MockOracleEngine
	future *products.Future
}

func testFuture(t *testing.T) *tstFuture {
	t.Helper()

	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)

	f := &types.Future{
		SettlementAsset: "ETH",
		QuoteName:       "ETH",
		OracleSpecForSettlementPrice: &types.OracleSpec{
			PubKeys: []string{"0xDEADBEEF"},
			Filters: []*types.OracleSpecFilter{
				{
					Key: &types.OracleSpecPropertyKey{
						Name: "price.ETH.value",
						Type: oraclespb.PropertyKey_TYPE_INTEGER,
					},
					Conditions: nil,
				},
			},
		},
		OracleSpecForTradingTermination: &types.OracleSpec{
			PubKeys: []string{"0xDEADBEEF"},
			Filters: []*types.OracleSpecFilter{
				{
					Key: &types.OracleSpecPropertyKey{
						Name: "trading.termination",
						Type: oraclespb.PropertyKey_TYPE_BOOLEAN,
					},
					Conditions: nil,
				},
			},
		},
		OracleSpecBinding: &types.OracleSpecBindingForFuture{
			SettlementPriceProperty:    "price.ETH.value",
			TradingTerminationProperty: "trading.termination",
		},
		SettlementPriceDecimals: 5,
	}

	oe.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(oracles.SubscriptionID(1))

	oe.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(oracles.SubscriptionID(2))

	future, err := products.NewFuture(context.Background(), log, f, oe)
	if err != nil {
		t.Fatalf("couldn't create a Future for testing: %v", err)
	}
	return &tstFuture{
		future: future,
		oe:     oe,
	}
}
