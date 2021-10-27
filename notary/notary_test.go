package notary_test

import (
	"context"
	"testing"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	bmock "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/notary"
	"code.vegaprotocol.io/vega/notary/mocks"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testNotary struct {
	// *notary.SnapshotNotary
	*notary.Notary
	ctrl   *gomock.Controller
	top    *mocks.MockValidatorTopology
	cmd    *mocks.MockCommander
	tt     *mocks.MockTimeTicker
	onTick func(context.Context, time.Time)
}

func getTestNotary(t *testing.T) *testNotary {
	t.Helper()
	ctrl := gomock.NewController(t)
	top := mocks.NewMockValidatorTopology(ctrl)
	broker := bmock.NewMockBroker(ctrl)
	cmd := mocks.NewMockCommander(ctrl)
	tt := mocks.NewMockTimeTicker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	var onTick func(context.Context, time.Time)
	// register the call back, will be needed during tests
	tt.EXPECT().NotifyOnTick(gomock.Any()).Times(1).Do(func(f func(context.Context, time.Time)) {
		onTick = f
	})
	// notr := notary.NewWithSnapshot(logging.NewTestLogger(), notary.NewDefaultConfig(), top, broker, cmd)
	notr := notary.New(
		logging.NewTestLogger(), notary.NewDefaultConfig(), top, broker, cmd, tt)
	return &testNotary{
		// SnapshotNotary: notr,
		Notary: notr,
		top:    top,
		ctrl:   ctrl,
		cmd:    cmd,
		onTick: onTick,
		tt:     tt,
	}
}

func TestNotary(t *testing.T) {
	t.Run("test add key for unknow resource - fail", testAddKeyForKOResource)
	t.Run("test add bad signature for known resource - success", testAddBadSignatureForOKResource)
	t.Run("test add key finalize all sig", testAddKeyFinalize)
}

func testAddKeyForKOResource(t *testing.T) {
	notr := getTestNotary(t)
	kind := types.NodeSignatureKindAssetNew
	resID := "resid"
	key := "123456"
	sig := []byte("123456")

	ns := commandspb.NodeSignature{
		Sig:  sig,
		Id:   resID,
		Kind: kind,
	}

	// first try to add a key for invalid resource
	err := notr.RegisterSignature(context.Background(), key, ns)
	assert.EqualError(t, err, notary.ErrUnknownResourceID.Error())

	// then try to start twice an aggregate
	notr.top.EXPECT().IsValidator().Times(1).Return(true)

	notr.StartAggregate(resID, kind, sig)
	assert.Panics(t, func() { notr.StartAggregate(resID, kind, sig) }, "expect to panic")
}

func testAddBadSignatureForOKResource(t *testing.T) {
	notr := getTestNotary(t)

	kind := types.NodeSignatureKindAssetNew
	resID := "resid"
	key := "123456"
	sig := []byte("123456")

	// start to aggregate, being a validator or not here doesn't matter
	notr.top.EXPECT().IsValidator().Times(1).Return(false)
	notr.StartAggregate(resID, kind, nil) // we send nil here if we are no validator

	ns := commandspb.NodeSignature{
		Sig:  sig,
		Id:   resID,
		Kind: kind,
	}

	// The signature we have received is not from a validator
	notr.top.EXPECT().IsValidatorVegaPubKey(gomock.Any()).AnyTimes().Return(false)

	err := notr.RegisterSignature(context.Background(), key, ns)
	assert.EqualError(t, err, notary.ErrNotAValidatorSignature.Error())
}

func testAddKeyFinalize(t *testing.T) {
	notr := getTestNotary(t)

	kind := types.NodeSignatureKindAssetNew
	resID := "resid"
	key := "123456"
	sig := []byte("123456")

	// add a valid node
	notr.top.EXPECT().Len().AnyTimes().Return(1)
	notr.top.EXPECT().IsValidatorVegaPubKey(gomock.Any()).AnyTimes().Return(true)

	notr.top.EXPECT().IsValidator().Times(1).Return(true)
	notr.StartAggregate(resID, kind, sig)

	// expect command to be send on next on time update
	notr.cmd.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	notr.onTick(context.Background(), time.Now())

	ns := commandspb.NodeSignature{
		Sig:  sig,
		Id:   resID,
		Kind: kind,
	}

	// first try to add a key for invalid resource
	notr.top.EXPECT().SelfVegaPubKey().Times(1).Return(key)
	err := notr.RegisterSignature(context.Background(), key, ns)
	assert.NoError(t, err, notary.ErrUnknownResourceID.Error())

	signatures, ok := notr.IsSigned(context.Background(), resID, kind)
	assert.True(t, ok)
	assert.Len(t, signatures, 1)
}
