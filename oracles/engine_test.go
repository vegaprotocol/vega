package oracles_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/oracles"
	oraclesv1 "code.vegaprotocol.io/vega/proto/oracles/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOracleEngine(t *testing.T) {
	t.Run("Subscribing to oracle engine succeeds", testOracleEngineSubscribingSucceeds)
	t.Run("Subscribing to oracle engine with without callback fails", testOracleEngineSubscribingWithoutCallbackFails)
	t.Run("Broadcasting to right callback with correct data succeeds", testOracleEngineBroadcastingCorrectDataSucceeds)
	t.Run("Broadcasting to right callback with incorrect data fails", testOracleEngineBroadcastingIncorrectDataFails)
	t.Run("Unsubscribing known ID from oracle engine succeeds", testOracleEngineUnsubscribingKnownIDSucceeds)
	t.Run("Unsubscribing unknown ID from oracle engine panics", testOracleEngineUnsubscribingUnknownIDPanics)
}

func testOracleEngineSubscribingSucceeds(t *testing.T) {
	// given
	btcEquals42 := spec("BTC", oraclesv1.Condition_OPERATOR_EQUALS, "42")
	ethLess84 := spec("ETH", oraclesv1.Condition_OPERATOR_LESS_THAN, "84")

	// setup
	engine := newEngine(t)

	// when
	id1 := engine.Subscribe(btcEquals42.Spec, btcEquals42.Subscriber.Cb)
	id2 := engine.Subscribe(ethLess84.Spec, ethLess84.Subscriber.Cb)

	// then
	assert.Equal(t, oracles.SubscriptionID(1), id1)
	assert.Equal(t, oracles.SubscriptionID(2), id2)
}

func testOracleEngineSubscribingWithoutCallbackFails(t *testing.T) {
	// given
	spec := spec("BTC", oraclesv1.Condition_OPERATOR_EQUALS, "42")

	// when
	subscribe := func() {
		newEngine(t).Subscribe(spec.Spec, nil)
	}

	// then
	assert.Panics(t, subscribe)
}

func testOracleEngineBroadcastingCorrectDataSucceeds(t *testing.T) {
	// given
	btcEquals42 := spec("BTC", oraclesv1.Condition_OPERATOR_EQUALS, "42")
	btcGreater21 := spec("BTC", oraclesv1.Condition_OPERATOR_GREATER_THAN, "21")
	ethEquals42 := spec("ETH", oraclesv1.Condition_OPERATOR_EQUALS, "42")
	ethLess84 := spec("ETH", oraclesv1.Condition_OPERATOR_LESS_THAN, "84")
	btcGreater100 := spec("BTC", oraclesv1.Condition_OPERATOR_GREATER_THAN, "100")
	dataBTC42 := dataWithPrice("BTC", "42")

	// setup
	engine := newEngine(t)

	// when
	engine.Subscribe(btcEquals42.Spec, btcEquals42.Subscriber.Cb)
	engine.Subscribe(ethEquals42.Spec, ethEquals42.Subscriber.Cb)
	engine.Subscribe(btcGreater21.Spec, btcGreater21.Subscriber.Cb)
	engine.Subscribe(ethLess84.Spec, ethLess84.Subscriber.Cb)
	engine.Subscribe(btcGreater100.Spec, btcGreater100.Subscriber.Cb)
	errB := engine.BroadcastData(context.Background(), dataBTC42)

	// then
	require.NoError(t, errB)
	assert.Equal(t, &dataBTC42, btcEquals42.Subscriber.ReceivedData)
	assert.Equal(t, &dataBTC42, btcEquals42.Subscriber.ReceivedData)
	assert.Nil(t, ethEquals42.Subscriber.ReceivedData)
	assert.Nil(t, ethLess84.Subscriber.ReceivedData)
	assert.Nil(t, btcGreater100.Subscriber.ReceivedData)
}

func testOracleEngineBroadcastingIncorrectDataFails(t *testing.T) {
	// given
	btcEquals42 := spec("BTC", oraclesv1.Condition_OPERATOR_EQUALS, "42")
	dataBTC42 := dataWithPrice("BTC", "hello")

	// setup
	engine := newEngine(t)

	// when
	_ = engine.Subscribe(btcEquals42.Spec, btcEquals42.Subscriber.Cb)
	errB := engine.BroadcastData(context.Background(), dataBTC42)

	// then
	assert.Error(t, errB)
	assert.Nil(t, btcEquals42.Subscriber.ReceivedData)
}

func testOracleEngineUnsubscribingUnknownIDPanics(t *testing.T) {
	// setup
	engine := newEngine(t)

	// when
	unsubscribe := func() {
		engine.Unsubscribe(oracles.SubscriptionID(1))
	}

	// then
	assert.Panics(t, unsubscribe)
}

func testOracleEngineUnsubscribingKnownIDSucceeds(t *testing.T) {
	// given
	btcEquals42 := spec("BTC", oraclesv1.Condition_OPERATOR_EQUALS, "42")
	ethEquals42 := spec("ETH", oraclesv1.Condition_OPERATOR_EQUALS, "42")
	dataBTC42 := dataWithPrice("BTC", "42")
	dataETH42 := dataWithPrice("ETH", "42")

	// setup
	engine := newEngine(t)

	// when
	idS1 := engine.Subscribe(btcEquals42.Spec, btcEquals42.Subscriber.Cb)
	engine.Subscribe(ethEquals42.Spec, ethEquals42.Subscriber.Cb)
	engine.Unsubscribe(idS1)
	errB1 := engine.BroadcastData(context.Background(), dataETH42)
	errB2 := engine.BroadcastData(context.Background(), dataBTC42)

	// then
	require.NoError(t, errB1)
	require.NoError(t, errB2)
	assert.Equal(t, &dataETH42, ethEquals42.Subscriber.ReceivedData)
	assert.Nil(t, btcEquals42.Subscriber.ReceivedData)
}

type testEngine struct {
	*oracles.Engine
}

func newEngine(_ *testing.T) *testEngine {
	return &testEngine{oracles.NewEngine()}
}

func dataWithPrice(currency, price string) oracles.OracleData {
	return oracles.OracleData{
		Data: map[string]string{
			fmt.Sprintf("prices.%s.value", currency): price,
		},
		PubKeys: []string{
			"0xCAFED00D",
		},
	}
}

type specBundle struct {
	Spec       oracles.OracleSpec
	Subscriber dummySubscriber
}

func spec(currency string, op oraclesv1.Condition_Operator, price string) specBundle {
	spec, _ := oracles.NewOracleSpec(oraclesv1.OracleSpec{
		PubKeys: []string{
			"0xCAFED00D",
		},
		Filters: []*oraclesv1.Filter{
			{
				Key: &oraclesv1.PropertyKey{
					Name: fmt.Sprintf("prices.%s.value", currency),
					Type: oraclesv1.PropertyKey_TYPE_INTEGER,
				},
				Conditions: []*oraclesv1.Condition{
					{
						Value:    price,
						Operator: op,
					},
				},
			},
		},
	})

	return specBundle{
		Spec:       *spec,
		Subscriber: dummySubscriber{},
	}
}

type dummySubscriber struct {
	ReceivedData *oracles.OracleData
}

func (d *dummySubscriber) Cb(_ context.Context, data oracles.OracleData) {
	d.ReceivedData = &data
}
