package notary_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/notary"
	"code.vegaprotocol.io/vega/notary/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testNotary struct {
	*notary.Notary
	ctrl   *gomock.Controller
	top    *mocks.MockValidatorTopology
	broker *mocks.MockBroker
}

func getTestNotary(t *testing.T) *testNotary {
	ctrl := gomock.NewController(t)
	top := mocks.NewMockValidatorTopology(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	notr := notary.New(logging.NewTestLogger(), notary.NewDefaultConfig(), top, broker)
	return &testNotary{
		Notary: notr,
		top:    top,
		ctrl:   ctrl,
	}
}

func TestNotary(t *testing.T) {
	t.Run("test add key for unknow resource - fail", testAddKeyForKOResource)
	t.Run("test add key for known resource - success", testAddKeyForOKResource)
	t.Run("test add key finalize all sig", testAddKeyFinalize)
}

func testAddKeyForKOResource(t *testing.T) {
	notr := getTestNotary(t)
	kind := types.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW
	resID := "resid"
	key := []byte(string("123456"))
	sig := []byte(string("123456"))

	ns := types.NodeSignature{
		Sig:  sig,
		ID:   resID,
		Kind: kind,
	}

	// first try to add a key for invalid resource
	_, ok, err := notr.AddSig(context.Background(), key, ns)
	assert.EqualError(t, err, notary.ErrUnknownResourceID.Error())
	assert.False(t, ok)

	// then try to start twice an aggregate
	err = notr.StartAggregate(resID, kind)
	assert.NoError(t, err)
	err = notr.StartAggregate(resID, kind)
	assert.EqualError(t, err, notary.ErrAggregateSigAlreadyStartedForResource.Error())
}

func testAddKeyForOKResource(t *testing.T) {
	notr := getTestNotary(t)

	kind := types.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW
	resID := "resid"
	key := []byte(string("123456"))
	sig := []byte(string("123456"))

	err := notr.StartAggregate(resID, kind)
	assert.NoError(t, err)

	notr.top.EXPECT().Exists(gomock.Any()).AnyTimes().Return(false)

	ns := types.NodeSignature{
		Sig:  sig,
		ID:   resID,
		Kind: kind,
	}

	// first try to add a key for invalid resource
	_, ok, err := notr.AddSig(context.Background(), key, ns)
	assert.EqualError(t, err, notary.ErrNotAValidatorSignature.Error())
	assert.False(t, ok)
}

func testAddKeyFinalize(t *testing.T) {
	notr := getTestNotary(t)

	kind := types.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW
	resID := "resid"
	key := []byte(string("123456"))
	sig := []byte(string("123456"))

	// add a valid node
	notr.top.EXPECT().Len().AnyTimes().Return(1)
	notr.top.EXPECT().Exists(gomock.Any()).AnyTimes().Return(true)

	err := notr.StartAggregate(resID, kind)
	assert.NoError(t, err)

	ns := types.NodeSignature{
		Sig:  sig,
		ID:   resID,
		Kind: kind,
	}

	// first try to add a key for invalid resource
	sigs, ok, err := notr.AddSig(context.Background(), key, ns)
	assert.NoError(t, err, notary.ErrUnknownResourceID.Error())
	assert.True(t, ok)
	assert.Len(t, sigs, 1)
}
