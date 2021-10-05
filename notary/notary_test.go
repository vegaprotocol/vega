package notary_test

import (
	"context"
	"testing"

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
	*notary.Notary
	ctrl *gomock.Controller
	top  *mocks.MockValidatorTopology
	cmd  *mocks.MockCommander
}

func getTestNotary(t *testing.T) *testNotary {
	ctrl := gomock.NewController(t)
	top := mocks.NewMockValidatorTopology(ctrl)
	broker := bmock.NewMockBroker(ctrl)
	cmd := mocks.NewMockCommander(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	notr := notary.New(logging.NewTestLogger(), notary.NewDefaultConfig(), top, broker, cmd)
	return &testNotary{
		Notary: notr,
		top:    top,
		ctrl:   ctrl,
		cmd:    cmd,
	}
}

func TestNotary(t *testing.T) {
	t.Run("test add key for unknow resource - fail", testAddKeyForKOResource)
	t.Run("test add key for known resource - success", testAddKeyForOKResource)
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
	_, ok, err := notr.AddSig(context.Background(), key, ns)
	assert.EqualError(t, err, notary.ErrUnknownResourceID.Error())
	assert.False(t, ok)

	// then try to start twice an aggregate
	notr.StartAggregate(resID, kind)
	assert.Panics(t, func() { notr.StartAggregate(resID, kind) }, "expect to panic")
}

func testAddKeyForOKResource(t *testing.T) {
	notr := getTestNotary(t)

	kind := types.NodeSignatureKindAssetNew
	resID := "resid"
	key := "123456"
	sig := []byte("123456")

	notr.StartAggregate(resID, kind)
	notr.top.EXPECT().IsValidatorVegaPubKey(gomock.Any()).AnyTimes().Return(false)

	ns := commandspb.NodeSignature{
		Sig:  sig,
		Id:   resID,
		Kind: kind,
	}

	// first try to add a key for invalid resource
	_, ok, err := notr.AddSig(context.Background(), key, ns)
	assert.EqualError(t, err, notary.ErrNotAValidatorSignature.Error())
	assert.False(t, ok)
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

	notr.StartAggregate(resID, kind)

	ns := commandspb.NodeSignature{
		Sig:  sig,
		Id:   resID,
		Kind: kind,
	}

	// first try to add a key for invalid resource
	sigs, ok, err := notr.AddSig(context.Background(), key, ns)
	assert.NoError(t, err, notary.ErrUnknownResourceID.Error())
	assert.True(t, ok)
	assert.Len(t, sigs, 1)
}
