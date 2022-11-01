// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package oracles_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	bmok "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/oracles/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOracleEngine(t *testing.T) {
	t.Run("Oracle listens to given public keys succeeds", testOracleEngineListensToSignersSucceeds)
	t.Run("Oracle listens to given public keys fails", testOracleEngineListensToSignersFails)
	t.Run("Subscribing to oracle engine succeeds", testOracleEngineSubscribingSucceeds)
	t.Run("Subscribing to oracle engine with without callback fails", testOracleEngineSubscribingWithoutCallbackFails)
	t.Run("Broadcasting to matching data succeeds", testOracleEngineBroadcastingMatchingDataSucceeds)
	t.Run("Unsubscribing known ID from oracle engine succeeds", testOracleEngineUnsubscribingKnownIDSucceeds)
	t.Run("Unsubscribing unknown ID from oracle engine panics", testOracleEngineUnsubscribingUnknownIDPanics)
	t.Run("Updating current time succeeds", testOracleEngineUpdatingCurrentTimeSucceeds)
}

func testOracleEngineListensToSignersSucceeds(t *testing.T) {
	// test conditions
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)

	// test oracle engine with 1 subscriber and 1 key provided
	btcEquals42 := spec(t, "BTC", datapb.Condition_OPERATOR_EQUALS, "42")
	engine.broker.expectNewOracleSpecSubscription(currentTime, btcEquals42.spec.OriginalSpec)
	_, _ = engine.Subscribe(ctx, btcEquals42.spec, btcEquals42.subscriber.Cb)

	// test oracle data with single PubKey
	data := oracles.OracleData{
		Signers: []*types.Signer{
			types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
		},
		Data: map[string]string{
			"my_key": "not an integer",
		},
	}

	result := engine.ListensToSigners(data)
	assert.True(t, result)

	// test oracle engine with 2 subscribers and multiple keys provided for one of them
	ethEquals42 := spec(t, "ETH", datapb.Condition_OPERATOR_LESS_THAN, "84", "0xCAFED00X", "0xCAFED00D", "0xBEARISH7", "0xBULLISH5")
	engine.broker.expectNewOracleSpecSubscription(currentTime, ethEquals42.spec.OriginalSpec)
	_, _ = engine.Subscribe(ctx, ethEquals42.spec, ethEquals42.subscriber.Cb)

	signersAppend := []*types.Signer{
		types.CreateSignerFromString("0xBEARISH7", types.DataSignerTypePubKey),
		types.CreateSignerFromString("0xBULLISH5", types.DataSignerTypePubKey),
	}

	data.Signers = append(data.Signers, signersAppend...)
	result = engine.ListensToSigners(data)
	assert.True(t, result)

	// test oracle data with 3 subscribers and multiple keys for some of them
	btcGreater21 := spec(t, "BTC", datapb.Condition_OPERATOR_GREATER_THAN, "21", "0xCAFED00D", "0xBEARISH7", "0xBULLISH5", "0xMILK123", "OxMILK456")
	engine.broker.expectNewOracleSpecSubscription(currentTime, btcGreater21.spec.OriginalSpec)
	_, _ = engine.Subscribe(ctx, btcGreater21.spec, btcGreater21.subscriber.Cb)

	data.Signers = append(data.Signers, types.CreateSignerFromString("0xMILK123", types.DataSignerTypePubKey))
	result = engine.ListensToSigners(data)
	assert.True(t, result)
}

func testOracleEngineListensToSignersFails(t *testing.T) {
	// test conditions
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)

	// test oracle engine with single subscriber and wrong key
	btcEquals42 := spec(t, "BTC", datapb.Condition_OPERATOR_EQUALS, "42", "0xWRONGKEY")
	engine.broker.expectNewOracleSpecSubscription(currentTime, btcEquals42.spec.OriginalSpec)
	_, _ = engine.Subscribe(ctx, btcEquals42.spec, btcEquals42.subscriber.Cb)

	data := oracles.OracleData{
		Signers: []*types.Signer{
			types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
			types.CreateSignerFromString("0xBEARISH17", types.DataSignerTypePubKey),
		},
		Data: map[string]string{
			"my_key": "not an integer",
		},
	}

	result := engine.ListensToSigners(data)
	assert.False(t, result)

	// test oracle engine with 2 subscribers and multiple missing keys
	ethEquals42 := spec(t, "ETH", datapb.Condition_OPERATOR_LESS_THAN, "84", "0xBEARISH7", "0xBULLISH5")
	engine.broker.expectNewOracleSpecSubscription(currentTime, ethEquals42.spec.OriginalSpec)
	_, _ = engine.Subscribe(ctx, ethEquals42.spec, ethEquals42.subscriber.Cb)

	signersAppend := []*types.Signer{
		types.CreateSignerFromString("0xMILK123", types.DataSignerTypePubKey),
		types.CreateSignerFromString("OxMILK456", types.DataSignerTypePubKey),
	}
	data.Signers = append(data.Signers, signersAppend...)
	result = engine.ListensToSigners(data)
	assert.False(t, result)
}

func testOracleEngineSubscribingSucceeds(t *testing.T) {
	// given
	btcEquals42 := spec(t, "BTC", datapb.Condition_OPERATOR_EQUALS, "42")
	ethLess84 := spec(t, "ETH", datapb.Condition_OPERATOR_LESS_THAN, "84")

	// setup
	ctx := context.Background()
	currentTime := time.Now()

	engine := newEngine(ctx, t, currentTime)
	engine.broker.expectNewOracleSpecSubscription(currentTime, btcEquals42.spec.OriginalSpec)
	engine.broker.expectNewOracleSpecSubscription(currentTime, ethLess84.spec.OriginalSpec)

	// when
	id1, _ := engine.Subscribe(ctx, btcEquals42.spec, btcEquals42.subscriber.Cb)
	id2, _ := engine.Subscribe(ctx, ethLess84.spec, ethLess84.subscriber.Cb)

	// then
	assert.Equal(t, oracles.SubscriptionID(1), id1)
	assert.Equal(t, oracles.SubscriptionID(2), id2)
}

func testOracleEngineSubscribingWithoutCallbackFails(t *testing.T) {
	// given
	spec := spec(t, "BTC", datapb.Condition_OPERATOR_EQUALS, "42")

	// setup
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)

	// when
	subscribe := func() {
		engine.Subscribe(ctx, spec.spec, nil)
	}

	// then
	assert.Panics(t, subscribe)
}

func testOracleEngineBroadcastingMatchingDataSucceeds(t *testing.T) {
	// given
	btcEquals42 := spec(t, "BTC", datapb.Condition_OPERATOR_EQUALS, "42")
	btcGreater21 := spec(t, "BTC", datapb.Condition_OPERATOR_GREATER_THAN, "21")
	ethEquals42 := spec(t, "ETH", datapb.Condition_OPERATOR_EQUALS, "42")
	ethLess84 := spec(t, "ETH", datapb.Condition_OPERATOR_LESS_THAN, "84")
	btcGreater100 := spec(t, "BTC", datapb.Condition_OPERATOR_GREATER_THAN, "100")
	dataBTC42 := dataWithPrice("BTC", "42")

	// setup
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)
	engine.broker.expectNewOracleSpecSubscription(currentTime, btcEquals42.spec.OriginalSpec)
	engine.broker.expectNewOracleSpecSubscription(currentTime, btcGreater21.spec.OriginalSpec)
	engine.broker.expectNewOracleSpecSubscription(currentTime, ethEquals42.spec.OriginalSpec)
	engine.broker.expectNewOracleSpecSubscription(currentTime, ethLess84.spec.OriginalSpec)
	engine.broker.expectNewOracleSpecSubscription(currentTime, btcGreater100.spec.OriginalSpec)
	engine.broker.expectMatchedOracleDataEvent(currentTime, &dataBTC42.proto, []string{
		btcEquals42.spec.OriginalSpec.ExternalDataSourceSpec.Spec.ID,
		btcGreater21.spec.OriginalSpec.ExternalDataSourceSpec.Spec.ID,
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
	btcEquals42 := spec(t, "BTC", datapb.Condition_OPERATOR_EQUALS, "42")
	ethEquals42 := spec(t, "ETH", datapb.Condition_OPERATOR_EQUALS, "42")
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)

	// expect
	engine.broker.expectNewOracleSpecSubscription(currentTime, btcEquals42.spec.OriginalSpec)

	// when
	idS1, _ := engine.Subscribe(ctx, btcEquals42.spec, btcEquals42.subscriber.Cb)

	// expect
	engine.broker.expectNewOracleSpecSubscription(currentTime, ethEquals42.spec.OriginalSpec)

	// when
	_, _ = engine.Subscribe(ctx, ethEquals42.spec, ethEquals42.subscriber.Cb)

	// expect
	engine.broker.expectOracleSpecSubscriptionDeactivation(currentTime, btcEquals42.spec.OriginalSpec)

	// when
	engine.Unsubscribe(ctx, idS1)

	// given
	dataETH42 := dataWithPrice("ETH", "42")

	// expect
	engine.broker.expectMatchedOracleDataEvent(currentTime, &dataETH42.proto, []string{
		ethEquals42.spec.OriginalSpec.ExternalDataSourceSpec.Spec.ID,
	})

	// when
	err := engine.BroadcastData(context.Background(), dataETH42.data)

	// then
	require.NoError(t, err)
	assert.Equal(t, &dataETH42.data, ethEquals42.subscriber.ReceivedData)
}

func testOracleEngineUpdatingCurrentTimeSucceeds(t *testing.T) {
	// setup
	ctx := context.Background()
	time30 := time.Unix(30, 0)
	time60 := time.Unix(60, 0)
	engine := newEngine(ctx, t, time30)
	assert.Equal(t, time30, engine.ts.GetTimeNow())

	engine2 := newEngine(ctx, t, time60)
	assert.Equal(t, time60, engine2.ts.GetTimeNow())
}

type testEngine struct {
	*oracles.Engine
	ts     *testTimeService
	broker *testBroker
}

// newEngine returns new Oracle test engine, but with preset time, so we can test against its value.
func newEngine(ctx context.Context, t *testing.T, tm time.Time) *testEngine {
	t.Helper()
	broker := newBroker(ctx, t)

	ts := newTimeService(ctx, t)
	ts.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return tm
		}).AnyTimes()

	te := &testEngine{
		Engine: oracles.NewEngine(
			logging.NewTestLogger(),
			oracles.NewDefaultConfig(),
			ts,
			broker,
		),
		ts:     ts,
		broker: broker,
	}

	return te
}

type dataBundle struct {
	data  oracles.OracleData
	proto datapb.OracleData
}

func dataWithPrice(currency, price string) dataBundle {
	priceName := fmt.Sprintf("prices.%s.value", currency)
	signers := []*types.Signer{
		types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey),
	}

	return dataBundle{
		data: oracles.OracleData{
			Data: map[string]string{
				priceName: price,
			},
			Signers: signers,
		},
		proto: datapb.OracleData{
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Data: []*datapb.Property{
						{
							Name:  priceName,
							Value: price,
						},
					},
					Signers: types.SignersIntoProto(signers),
				},
			},
		},
	}
}

type specBundle struct {
	spec       oracles.OracleSpec
	subscriber dummySubscriber
}

func spec(t *testing.T, currency string, op datapb.Condition_Operator, price string, keys ...string) specBundle {
	t.Helper()
	var signers []*types.Signer
	if len(keys) > 0 {
		signers = make([]*types.Signer, len(keys))
		for i, k := range keys {
			signers[i] = types.CreateSignerFromString(k, types.DataSignerTypePubKey)
		}
	}
	if len(keys) == 0 {
		signers = []*types.Signer{types.CreateSignerFromString("0xCAFED00D", types.DataSignerTypePubKey)}
	}

	typedOracleSpec := types.OracleSpecFromProto(
		&datapb.OracleSpec{
			ExternalDataSourceSpec: &datapb.ExternalDataSourceSpec{
				Spec: datapb.NewDataSourceSpec(
					&datapb.DataSourceSpecConfiguration{
						Signers: types.SignersIntoProto(signers),
						Filters: []*datapb.Filter{
							{
								Key: &datapb.PropertyKey{
									Name: fmt.Sprintf("prices.%s.value", currency),
									Type: datapb.PropertyKey_TYPE_INTEGER,
								},
								Conditions: []*datapb.Condition{
									{
										Value:    price,
										Operator: op,
									},
								},
							},
						},
					},
				),
			},
		})
	spec, err := oracles.NewOracleSpec(*typedOracleSpec.ExternalDataSourceSpec)
	if err != nil {
		t.Fatalf("Couldn't create oracle spec: %v", err)
	}
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
	*bmok.MockBroker
	ctx context.Context
}

type testTimeService struct {
	*mocks.MockTimeService
	ctx context.Context
}

func newBroker(ctx context.Context, t *testing.T) *testBroker {
	t.Helper()
	ctrl := gomock.NewController(t)
	return &testBroker{
		MockBroker: bmok.NewMockBroker(ctrl),
		ctx:        ctx,
	}
}

func newTimeService(ctx context.Context, t *testing.T) *testTimeService {
	t.Helper()
	ctrl := gomock.NewController(t)
	return &testTimeService{
		MockTimeService: mocks.NewMockTimeService(ctrl),
		ctx:             ctx,
	}
}

func (b *testBroker) expectNewOracleSpecSubscription(currentTime time.Time, spec *types.OracleSpec) {
	proto := spec.ExternalDataSourceSpec.IntoProto()
	proto.Spec.CreatedAt = currentTime.UnixNano()
	proto.Spec.Status = datapb.DataSourceSpec_STATUS_ACTIVE
	b.EXPECT().Send(events.NewOracleSpecEvent(b.ctx, datapb.OracleSpec{ExternalDataSourceSpec: proto})).Times(1)
}

func (b *testBroker) expectOracleSpecSubscriptionDeactivation(currentTime time.Time, spec *types.OracleSpec) {
	proto := spec.ExternalDataSourceSpec.IntoProto()
	proto.Spec.CreatedAt = currentTime.UnixNano()
	proto.Spec.Status = datapb.DataSourceSpec_STATUS_DEACTIVATED
	b.EXPECT().Send(events.NewOracleSpecEvent(b.ctx, datapb.OracleSpec{ExternalDataSourceSpec: proto})).Times(1)
}

func (b *testBroker) expectMatchedOracleDataEvent(currentTime time.Time, data *datapb.OracleData, specIDs []string) {
	data.ExternalData.Data.MatchedSpecIds = specIDs
	data.ExternalData.Data.BroadcastAt = currentTime.UnixNano()
	b.EXPECT().Send(events.NewOracleDataEvent(b.ctx, *data)).Times(1)
}
