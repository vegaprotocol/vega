package oracles_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/oracles/mocks"
	oraclespb "code.vegaprotocol.io/vega/proto/oracles/v1"

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
	t.Run("Updating current time succeeds", testOracleEngineUpdatingCurrentTimeSucceeds)
}

func testOracleEngineSubscribingSucceeds(t *testing.T) {
	// given
	btcEquals42 := spec("BTC", oraclespb.Condition_OPERATOR_EQUALS, "42")
	ethLess84 := spec("ETH", oraclespb.Condition_OPERATOR_LESS_THAN, "84")

	// setup
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)
	engine.broker.mockNewOracleSpecSubscription(currentTime, btcEquals42.spec.Proto)
	engine.broker.mockNewOracleSpecSubscription(currentTime, ethLess84.spec.Proto)

	// when
	id1 := engine.Subscribe(ctx, btcEquals42.spec, btcEquals42.subscriber.Cb)
	id2 := engine.Subscribe(ctx, ethLess84.spec, ethLess84.subscriber.Cb)

	// then
	assert.Equal(t, oracles.SubscriptionID(1), id1)
	assert.Equal(t, oracles.SubscriptionID(2), id2)
}

func testOracleEngineSubscribingWithoutCallbackFails(t *testing.T) {
	// given
	spec := spec("BTC", oraclespb.Condition_OPERATOR_EQUALS, "42")

	// setup
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)
	engine.broker.mockNewOracleSpecSubscription(currentTime, spec.spec.Proto)

	// when
	subscribe := func() {
		engine.Subscribe(ctx, spec.spec, nil)
	}

	// then
	assert.Panics(t, subscribe)
}

func testOracleEngineBroadcastingCorrectDataSucceeds(t *testing.T) {
	// given
	btcEquals42 := spec("BTC", oraclespb.Condition_OPERATOR_EQUALS, "42")
	btcGreater21 := spec("BTC", oraclespb.Condition_OPERATOR_GREATER_THAN, "21")
	ethEquals42 := spec("ETH", oraclespb.Condition_OPERATOR_EQUALS, "42")
	ethLess84 := spec("ETH", oraclespb.Condition_OPERATOR_LESS_THAN, "84")
	btcGreater100 := spec("BTC", oraclespb.Condition_OPERATOR_GREATER_THAN, "100")
	dataBTC42 := dataWithPrice("BTC", "42")

	// setup
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)
	engine.broker.mockNewOracleSpecSubscription(currentTime, btcEquals42.spec.Proto)
	engine.broker.mockNewOracleSpecSubscription(currentTime, btcGreater21.spec.Proto)
	engine.broker.mockNewOracleSpecSubscription(currentTime, ethEquals42.spec.Proto)
	engine.broker.mockNewOracleSpecSubscription(currentTime, ethLess84.spec.Proto)
	engine.broker.mockNewOracleSpecSubscription(currentTime, btcGreater100.spec.Proto)
	engine.broker.mockOracleDataBroadcast(currentTime, dataBTC42.proto, []string{
		btcEquals42.spec.Proto.Id,
		btcGreater21.spec.Proto.Id,
	})

	// when
	engine.Subscribe(ctx, btcEquals42.spec, btcEquals42.subscriber.Cb)
	engine.Subscribe(ctx, ethEquals42.spec, ethEquals42.subscriber.Cb)
	engine.Subscribe(ctx, btcGreater21.spec, btcGreater21.subscriber.Cb)
	engine.Subscribe(ctx, ethLess84.spec, ethLess84.subscriber.Cb)
	engine.Subscribe(ctx, btcGreater100.spec, btcGreater100.subscriber.Cb)
	errB := engine.BroadcastData(context.Background(), dataBTC42.data)

	// then
	require.NoError(t, errB)
	assert.Equal(t, &dataBTC42.data, btcEquals42.subscriber.ReceivedData)
	assert.Equal(t, &dataBTC42.data, btcGreater21.subscriber.ReceivedData)
	assert.Nil(t, ethEquals42.subscriber.ReceivedData)
	assert.Nil(t, ethLess84.subscriber.ReceivedData)
	assert.Nil(t, btcGreater100.subscriber.ReceivedData)
}

func testOracleEngineBroadcastingIncorrectDataFails(t *testing.T) {
	// given
	btcEquals42 := spec("BTC", oraclespb.Condition_OPERATOR_EQUALS, "42")
	dataBTC42 := dataWithPrice("BTC", "hello")

	// setup
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)
	engine.broker.mockNewOracleSpecSubscription(currentTime, btcEquals42.spec.Proto)

	// when
	_ = engine.Subscribe(ctx, btcEquals42.spec, btcEquals42.subscriber.Cb)
	errB := engine.BroadcastData(ctx, dataBTC42.data)

	// then
	assert.Error(t, errB)
	assert.Nil(t, btcEquals42.subscriber.ReceivedData)
}

func testOracleEngineUnsubscribingUnknownIDPanics(t *testing.T) {
	// setup
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)

	// when
	unsubscribe := func() {
		engine.Unsubscribe(ctx, oracles.SubscriptionID(1))
	}

	// then
	assert.Panics(t, unsubscribe)
}

func testOracleEngineUnsubscribingKnownIDSucceeds(t *testing.T) {
	// given
	btcEquals42 := spec("BTC", oraclespb.Condition_OPERATOR_EQUALS, "42")
	ethEquals42 := spec("ETH", oraclespb.Condition_OPERATOR_EQUALS, "42")
	dataBTC42 := dataWithPrice("BTC", "42")
	dataETH42 := dataWithPrice("ETH", "42")

	// setup
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)
	engine.broker.mockNewOracleSpecSubscription(currentTime, btcEquals42.spec.Proto)
	engine.broker.mockNewOracleSpecSubscription(currentTime, ethEquals42.spec.Proto)
	engine.broker.mockOracleSpecSubscriptionDeactivation(currentTime, btcEquals42.spec.Proto)
	engine.broker.mockOracleDataBroadcast(currentTime, dataETH42.proto, []string{
		ethEquals42.spec.Proto.Id,
	})

	// when
	idS1 := engine.Subscribe(ctx, btcEquals42.spec, btcEquals42.subscriber.Cb)
	engine.Subscribe(ctx, ethEquals42.spec, ethEquals42.subscriber.Cb)
	engine.Unsubscribe(ctx, idS1)
	errB1 := engine.BroadcastData(context.Background(), dataETH42.data)
	errB2 := engine.BroadcastData(context.Background(), dataBTC42.data)

	// then
	require.NoError(t, errB1)
	require.NoError(t, errB2)
	assert.Equal(t, &dataETH42.data, ethEquals42.subscriber.ReceivedData)
	assert.Nil(t, btcEquals42.subscriber.ReceivedData)
}

func testOracleEngineUpdatingCurrentTimeSucceeds(t *testing.T) {
	// setup
	ctx := context.Background()
	time30 := time.Unix(30, 0)
	time60 := time.Unix(60, 0)
	engine := newEngine(ctx, t, time30)

	// when
	engine.UpdateCurrentTime(ctx, time60)

	// then
	assert.Equal(t, time60, engine.CurrentTime)
}

type testEngine struct {
	*oracles.Engine
	broker *testBroker
}

func newEngine(ctx context.Context, t *testing.T, currentTime time.Time) *testEngine {
	broker := newBroker(ctx, t)
	return &testEngine{
		Engine: oracles.NewEngine(
			logging.NewTestLogger(),
			oracles.NewDefaultConfig(),
			currentTime,
			broker,
		),
		broker: broker,
	}
}

type dataBundle struct {
	data  oracles.OracleData
	proto oraclespb.OracleData
}

func dataWithPrice(currency, price string) dataBundle {
	priceName := fmt.Sprintf("prices.%s.value", currency)
	return dataBundle{
		data: oracles.OracleData{
			Data: map[string]string{
				priceName: price,
			},
			PubKeys: []string{
				"0xCAFED00D",
			},
		},
		proto: oraclespb.OracleData{
			Data: []*oraclespb.Property{
				{
					Name:  priceName,
					Value: price,
				},
			},
			PubKeys: []string{
				"0xCAFED00D",
			},
		},
	}
}

type specBundle struct {
	spec       oracles.OracleSpec
	subscriber dummySubscriber
}

func spec(currency string, op oraclespb.Condition_Operator, price string) specBundle {
	spec, _ := oracles.NewOracleSpec(*oraclespb.NewOracleSpec(
		[]string{
			"0xCAFED00D",
		},
		[]*oraclespb.Filter{
			{
				Key: &oraclespb.PropertyKey{
					Name: fmt.Sprintf("prices.%s.value", currency),
					Type: oraclespb.PropertyKey_TYPE_INTEGER,
				},
				Conditions: []*oraclespb.Condition{
					{
						Value:    price,
						Operator: op,
					},
				},
			},
		}))
	return specBundle{
		spec:       *spec,
		subscriber: dummySubscriber{},
	}
}

type dummySubscriber struct {
	ReceivedData *oracles.OracleData
}

func (d *dummySubscriber) Cb(_ context.Context, data oracles.OracleData) error {
	d.ReceivedData = &data
	return nil
}

type testBroker struct {
	*mocks.MockBroker
	ctx context.Context
}

func newBroker(ctx context.Context, t *testing.T) *testBroker {
	ctrl := gomock.NewController(t)
	return &testBroker{
		MockBroker: mocks.NewMockBroker(ctrl),
		ctx: ctx,
	}
}

func (b *testBroker) mockNewOracleSpecSubscription(currentTime time.Time, spec oraclespb.OracleSpec) {
	spec.CreatedAt = currentTime.Unix()
	spec.Status = oraclespb.OracleSpec_STATUS_ACTIVE
	b.EXPECT().Send(events.NewOracleSpecEvent(b.ctx, spec))
}

func (b *testBroker) mockOracleSpecSubscriptionDeactivation(currentTime time.Time, spec oraclespb.OracleSpec) {
	spec.CreatedAt = currentTime.Unix()
	spec.Status = oraclespb.OracleSpec_STATUS_DEACTIVATED
	b.EXPECT().Send(events.NewOracleSpecEvent(b.ctx, spec))
}

func (b *testBroker) mockOracleDataBroadcast(currentTime time.Time, data oraclespb.OracleData, specIDs []string) {
	data.MatchedSpecIds = specIDs
	data.BroadcastAt = currentTime.Unix()
	b.EXPECT().Send(events.NewOracleDataEvent(b.ctx, data))
}
