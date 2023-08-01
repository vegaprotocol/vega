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

package spec_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	bmok "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/common"
	dsspec "code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/datasource/spec/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
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
	t.Run("Subscribing to oracle spec activation succeeds", testOracleEngineSubscribingToSpecActivationSucceeds)
}

func testOracleEngineListensToSignersSucceeds(t *testing.T) {
	// test conditions
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)

	// test oracle engine with 1 subscriber and 1 key provided
	btcEquals42 := spec(t, "BTC", datapb.Condition_OPERATOR_EQUALS, "42")
	engine.broker.expectNewSpecSubscription(currentTime, btcEquals42.spec.OriginalSpec)
	_, _, _ = engine.Subscribe(ctx, btcEquals42.spec, btcEquals42.subscriber.Cb)

	// test oracle data with single PubKey
	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xCAFED00D", common.SignerTypePubKey),
		},
		Data: map[string]string{
			"my_key": "not an integer",
		},
	}

	result := engine.ListensToSigners(data)
	assert.True(t, result)

	// test oracle engine with 2 subscribers and multiple keys provided for one of them
	ethEquals42 := spec(t, "ETH", datapb.Condition_OPERATOR_LESS_THAN, "84", "0xCAFED00X", "0xCAFED00D", "0xBEARISH7", "0xBULLISH5")
	engine.broker.expectNewSpecSubscription(currentTime, ethEquals42.spec.OriginalSpec)
	_, _, _ = engine.Subscribe(ctx, ethEquals42.spec, ethEquals42.subscriber.Cb)

	signersAppend := []*common.Signer{
		common.CreateSignerFromString("0xBEARISH7", common.SignerTypePubKey),
		common.CreateSignerFromString("0xBULLISH5", common.SignerTypePubKey),
	}

	data.Signers = append(data.Signers, signersAppend...)
	result = engine.ListensToSigners(data)
	assert.True(t, result)

	// test oracle data with 3 subscribers and multiple keys for some of them
	btcGreater21 := spec(t, "BTC", datapb.Condition_OPERATOR_GREATER_THAN, "21", "0xCAFED00D", "0xBEARISH7", "0xBULLISH5", "0xMILK123", "OxMILK456")
	engine.broker.expectNewSpecSubscription(currentTime, btcGreater21.spec.OriginalSpec)
	_, _, _ = engine.Subscribe(ctx, btcGreater21.spec, btcGreater21.subscriber.Cb)

	data.Signers = append(data.Signers, common.CreateSignerFromString("0xMILK123", common.SignerTypePubKey))
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
	engine.broker.expectNewSpecSubscription(currentTime, btcEquals42.spec.OriginalSpec)
	_, _, _ = engine.Subscribe(ctx, btcEquals42.spec, btcEquals42.subscriber.Cb)

	data := common.Data{
		Signers: []*common.Signer{
			common.CreateSignerFromString("0xCAFED00D", common.SignerTypePubKey),
			common.CreateSignerFromString("0xBEARISH17", common.SignerTypePubKey),
		},
		Data: map[string]string{
			"my_key": "not an integer",
		},
		MetaData: map[string]string{},
	}

	result := engine.ListensToSigners(data)
	assert.False(t, result)

	// test oracle engine with 2 subscribers and multiple missing keys
	ethEquals42 := spec(t, "ETH", datapb.Condition_OPERATOR_LESS_THAN, "84", "0xBEARISH7", "0xBULLISH5")
	engine.broker.expectNewSpecSubscription(currentTime, ethEquals42.spec.OriginalSpec)
	_, _, _ = engine.Subscribe(ctx, ethEquals42.spec, ethEquals42.subscriber.Cb)

	signersAppend := []*common.Signer{
		common.CreateSignerFromString("0xMILK123", common.SignerTypePubKey),
		common.CreateSignerFromString("OxMILK456", common.SignerTypePubKey),
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
	engine.broker.expectNewSpecSubscription(currentTime, btcEquals42.spec.OriginalSpec)
	engine.broker.expectNewSpecSubscription(currentTime, ethLess84.spec.OriginalSpec)

	// when
	id1, _, _ := engine.Subscribe(ctx, btcEquals42.spec, btcEquals42.subscriber.Cb)
	id2, _, _ := engine.Subscribe(ctx, ethLess84.spec, ethLess84.subscriber.Cb)

	// then
	assert.Equal(t, dsspec.SubscriptionID(1), id1)
	assert.Equal(t, dsspec.SubscriptionID(2), id2)
}

func testOracleEngineSubscribingToSpecActivationSucceeds(t *testing.T) {
	// given
	btcEquals42 := spec(t, "BTC", datapb.Condition_OPERATOR_EQUALS, "42")
	ethLess84 := spec(t, "ETH", datapb.Condition_OPERATOR_LESS_THAN, "84")

	subscriber1 := dummySubscriber{}
	subscriber2 := dummySubscriber{}

	// setup
	ctx := context.Background()
	currentTime := time.Now()

	subscriber := newTestActivationSubscriber()

	engine := newEngine(ctx, t, currentTime)
	engine.AddSpecActivationListener(subscriber)

	engine.broker.expectNewSpecSubscription(currentTime, btcEquals42.spec.OriginalSpec)

	id1, _, _ := engine.Subscribe(ctx, btcEquals42.spec, subscriber1.Cb)

	engine.broker.expectNewSpecSubscription(currentTime, ethLess84.spec.OriginalSpec)
	id2, _, _ := engine.Subscribe(ctx, ethLess84.spec, subscriber1.Cb)

	engine.broker.expectNewSpecSubscription(currentTime, ethLess84.spec.OriginalSpec)
	id3, _, _ := engine.Subscribe(ctx, ethLess84.spec, subscriber2.Cb)

	assert.Equal(t, 2, len(subscriber.activeSpecs))

	engine.Unsubscribe(ctx, id3)

	assert.Equal(t, 2, len(subscriber.activeSpecs))

	engine.broker.expectSpecSubscriptionDeactivation(currentTime, ethLess84.spec.OriginalSpec)
	engine.Unsubscribe(ctx, id2)
	assert.Equal(t, 1, len(subscriber.activeSpecs))

	engine.broker.expectSpecSubscriptionDeactivation(currentTime, btcEquals42.spec.OriginalSpec)
	engine.Unsubscribe(ctx, id1)
	assert.Equal(t, 0, len(subscriber.activeSpecs))
}

type testActivationSubscriber struct {
	activeSpecs map[string]datasource.Spec
}

func newTestActivationSubscriber() testActivationSubscriber {
	return testActivationSubscriber{activeSpecs: make(map[string]datasource.Spec)}
}

func (t testActivationSubscriber) OnSpecActivated(ctx context.Context, oracleSpec datasource.Spec) error {
	t.activeSpecs[oracleSpec.ID] = oracleSpec
	return nil
}

func (t testActivationSubscriber) OnSpecDeactivated(ctx context.Context, oracleSpec datasource.Spec) {
	delete(t.activeSpecs, oracleSpec.ID)
}

func testOracleEngineSubscribingWithoutCallbackFails(t *testing.T) {
	// given
	s := spec(t, "BTC", datapb.Condition_OPERATOR_EQUALS, "42")

	// setup
	ctx := context.Background()
	currentTime := time.Now()
	engine := newEngine(ctx, t, currentTime)

	// when
	subscribe := func() {
		engine.Subscribe(ctx, s.spec, nil)
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
	engine.broker.expectNewSpecSubscription(currentTime, btcEquals42.spec.OriginalSpec)
	engine.broker.expectNewSpecSubscription(currentTime, btcGreater21.spec.OriginalSpec)
	engine.broker.expectNewSpecSubscription(currentTime, ethEquals42.spec.OriginalSpec)
	engine.broker.expectNewSpecSubscription(currentTime, ethLess84.spec.OriginalSpec)
	engine.broker.expectNewSpecSubscription(currentTime, btcGreater100.spec.OriginalSpec)
	engine.broker.expectMatchedDataEvent(currentTime, &dataBTC42.proto, []string{
		btcEquals42.spec.OriginalSpec.ID,
		btcGreater21.spec.OriginalSpec.ID,
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
		engine.Unsubscribe(ctx, dsspec.SubscriptionID(1))
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
	engine.broker.expectNewSpecSubscription(currentTime, btcEquals42.spec.OriginalSpec)

	// when
	idS1, _, _ := engine.Subscribe(ctx, btcEquals42.spec, btcEquals42.subscriber.Cb)

	// expect
	engine.broker.expectNewSpecSubscription(currentTime, ethEquals42.spec.OriginalSpec)

	// when
	_, _, _ = engine.Subscribe(ctx, ethEquals42.spec, ethEquals42.subscriber.Cb)

	// expect
	engine.broker.expectSpecSubscriptionDeactivation(currentTime, btcEquals42.spec.OriginalSpec)

	// when
	engine.Unsubscribe(ctx, idS1)

	// given
	dataETH42 := dataWithPrice("ETH", "42")

	// expect
	engine.broker.expectMatchedDataEvent(currentTime, &dataETH42.proto, []string{
		ethEquals42.spec.OriginalSpec.ID,
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
	*dsspec.Engine
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
		Engine: dsspec.NewEngine(
			logging.NewTestLogger(),
			dsspec.NewDefaultConfig(),
			ts,
			broker,
		),
		ts:     ts,
		broker: broker,
	}

	return te
}

type dataBundle struct {
	data  common.Data
	proto vegapb.OracleData
}

func dataWithPrice(currency, price string) dataBundle {
	priceName := fmt.Sprintf("prices.%s.value", currency)
	signers := []*common.Signer{
		common.CreateSignerFromString("0xCAFED00D", common.SignerTypePubKey),
	}

	return dataBundle{
		data: common.Data{
			Data: map[string]string{
				priceName: price,
			},
			Signers: signers,
		},
		proto: vegapb.OracleData{
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Data: []*datapb.Property{
						{
							Name:  priceName,
							Value: price,
						},
					},
					MetaData: []*datapb.Property{},
					Signers:  common.SignersIntoProto(signers),
				},
			},
		},
	}
}

type specBundle struct {
	spec       dsspec.Spec
	subscriber dummySubscriber
}

func spec(t *testing.T, currency string, op datapb.Condition_Operator, price string, keys ...string) specBundle {
	t.Helper()
	var signers []*datapb.Signer
	if len(keys) > 0 {
		signers = make([]*datapb.Signer, len(keys))
		for i, k := range keys {
			signers[i] = &datapb.Signer{
				Signer: &datapb.Signer_PubKey{
					PubKey: &datapb.PubKey{
						Key: k,
					},
				},
			}
		}
	}
	if len(keys) == 0 {
		signers = []*datapb.Signer{
			{
				Signer: &datapb.Signer_PubKey{
					PubKey: &datapb.PubKey{
						Key: "0xCAFED00D",
					},
				},
			},
		}
	}

	testSpec := vegapb.NewDataSourceSpec(
		vegapb.NewDataSourceDefinition(
			vegapb.DataSourceContentTypeOracle,
		).SetOracleConfig(
			&vegapb.DataSourceDefinitionExternal_Oracle{
				Oracle: &vegapb.DataSourceSpecConfiguration{
					Signers: signers,
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
			},
		),
	)

	typedOracleSpec := datasource.SpecFromProto(testSpec)

	spec, err := dsspec.New(*typedOracleSpec)
	if err != nil {
		t.Fatalf("Couldn't create oracle spec: %v", err)
	}
	return specBundle{
		spec:       *spec,
		subscriber: dummySubscriber{},
	}
}

type dummySubscriber struct {
	ReceivedData *common.Data
}

func (d *dummySubscriber) Cb(_ context.Context, data common.Data) error {
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

func (b *testBroker) expectNewSpecSubscription(currentTime time.Time, spec *datasource.Spec) {
	proto := spec.IntoProto()
	proto.CreatedAt = currentTime.UnixNano()
	proto.Status = vegapb.DataSourceSpec_STATUS_ACTIVE
	b.EXPECT().Send(events.NewOracleSpecEvent(b.ctx, &vegapb.OracleSpec{ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{Spec: proto}})).Times(1)
}

func (b *testBroker) expectSpecSubscriptionDeactivation(currentTime time.Time, spec *datasource.Spec) {
	proto := spec.IntoProto()
	proto.CreatedAt = currentTime.UnixNano()
	proto.Status = vegapb.DataSourceSpec_STATUS_DEACTIVATED
	b.EXPECT().Send(events.NewOracleSpecEvent(b.ctx, &vegapb.OracleSpec{ExternalDataSourceSpec: &vegapb.ExternalDataSourceSpec{Spec: proto}})).Times(1)
}

func (b *testBroker) expectMatchedDataEvent(currentTime time.Time, data *vegapb.OracleData, specIDs []string) {
	data.ExternalData.Data.MatchedSpecIds = specIDs
	data.ExternalData.Data.BroadcastAt = currentTime.UnixNano()
	b.EXPECT().Send(events.NewOracleDataEvent(b.ctx, *data)).Times(1)
}
