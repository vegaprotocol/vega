package products_test

import (
	"context"
	"testing"

	v1 "code.vegaprotocol.io/protos/vega/oracles/v1"
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
	ft := testFuture(t, 5)

	// scaling factor = 5, asset scaling = 5
	// scaled = 123.45678 * 1 => to uint => 123
	scaled, overflow := ft.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("123.45678"), 5)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(123), scaled)

	// scaling factor = 5, asset scaling = 5
	// scaled = 1000000 * 1 => to uint => 1000000
	scaled, overflow = ft.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("1000000"), 5)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(1000000), scaled)

	// scaling factor = 5, asset scaling = 3
	// scaled = 123.45678 * 10^(3-5) => 1.2345678 => to uint => 1
	scaled, overflow = ft.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("123.45678"), 3)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(1), scaled)

	// scaling factor = 5, asset scaling = 3
	// scaled = 1000000 * 10^(3-5) => 10000 => to uint => 10000
	scaled, overflow = ft.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("1000000"), 3)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(10000), scaled)

	// scaling factor = 5, asset scaling = 7
	// scaled = 123.45678 * 10^(7-5) => 12345.678 => to uint => 12345
	scaled, overflow = ft.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("123.45678"), 7)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(12345), scaled)

	// scaling factor = 5, asset scaling = 7
	// scaled = 1000000 * 10^(7-5) => 100000000 => to uint => 100000000
	scaled, overflow = ft.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("1000000"), 7)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(100000000), scaled)

	// scaling factor = 5, asset scaling = 0
	// scaled = 123456.78 * 10^(0-5) => 1.2345678 => to uint => 1
	scaled, overflow = ft.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("123456.78"), 0)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(1), scaled)

	// scaling factor = 5, asset scaling = 0
	// scaled = 1000000 * 10^(0-5) => 10 => to uint => 1
	scaled, overflow = ft.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("1000000"), 0)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(10), scaled)

	tf := testFuture(t, 0)

	// scaling factor = 0, asset scaling = 0
	// scaled = 123456.78 * 10^(0-0) => 123456.78 => to uint => 123456
	scaled, overflow = tf.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("123456.78"), 0)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(123456), scaled)

	// scaling factor = 0, asset scaling = 0
	// scaled = 1000000 * 10^(0-0) => 1000000 => to uint => 1000000
	scaled, overflow = tf.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("1000000"), 0)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(1000000), scaled)

	// scaling factor = 0, asset scaling = 5
	// scaled = 123456.78 * 10^(5-0) => 12345678000 => to uint => 12345678000
	scaled, overflow = tf.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("123456.78"), 5)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(12345678000), scaled)

	// scaling factor = 0, asset scaling = 5
	// scaled = 1000000 * 10^(5-0) => 100000000000 => to uint => 100000000000
	scaled, overflow = tf.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("1000000"), 5)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(100000000000), scaled)

	// negative scaling
	ntf := testFuture(t, -2)

	// scaling factor = -2, asset scaling = 0
	// scaled = 123456.78 * 10^(0--2) => 12345678 => to uint => 12345678
	scaled, overflow = ntf.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("123456.78"), 0)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(12345678), scaled)

	// scaling factor = -2, asset scaling = 0
	// scaled = 1000000 * 10^(0--2) => 100000000 => to uint => 100000000
	scaled, overflow = ntf.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("1000000"), 0)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(100000000), scaled)

	// scaling factor = -2, asset scaling = 5
	// scaled = 123456.78 * 10^(5--2) => 1234567800000 => to uint => 1234567800000
	scaled, overflow = ntf.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("123456.78"), 5)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(1234567800000), scaled)

	// scaling factor = -2, asset scaling = 5
	// scaled = 1000000 * 10^(5--2) => 10000000000000 => to uint => 10000000000000
	scaled, overflow = ntf.future.ScaleSettlementPriceToDecimalPlaces(num.MustDecimalFromString("1000000"), 5)
	require.Equal(t, false, overflow)
	require.Equal(t, num.NewUint(10000000000000), scaled)
}

func testUnsubscribingTheOracleEngineSucceeds(t *testing.T) {
	// given
	tf := testFuture(t, 0)
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

func testFuture(t *testing.T, scaling int32) *tstFuture {
	t.Helper()

	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)

	f := &types.Future{
		SettlementAsset: "ETH",
		QuoteName:       "ETH",
		OracleSpecForSettlementPrice: &v1.OracleSpec{
			PubKeys: []string{"0xDEADBEEF"},
			Filters: []*v1.Filter{
				{
					Key: &v1.PropertyKey{
						Name: "price.ETH.value",
						Type: v1.PropertyKey_TYPE_INTEGER,
					},
					Conditions: nil,
				},
			},
		},
		OracleSpecForTradingTermination: &v1.OracleSpec{
			PubKeys: []string{"0xDEADBEEF"},
			Filters: []*v1.Filter{
				{
					Key: &v1.PropertyKey{
						Name: "trading.termination",
						Type: v1.PropertyKey_TYPE_BOOLEAN,
					},
					Conditions: nil,
				},
			},
		},
		OracleSpecBinding: &types.OracleSpecToFutureBinding{
			SettlementPriceProperty:    "price.ETH.value",
			TradingTerminationProperty: "trading.termination",
		},
		SettlementPriceDecimalScalingExponent: scaling,
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
