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
	"github.com/golang/mock/gomock"
)

func TestFuture(t *testing.T) {
	t.Run("Unsubscribing the oracle engine succeeds", testUnsubscribingTheOracleEngineSucceeds)
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
