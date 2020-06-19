package notary_test

import (
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/notary"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestNotary(t *testing.T) {
	t.Run("test add key for unknow resource - fail", testAddKeyForKOResource)
	t.Run("test add key for known resource - success", testAddKeyForOKResource)
	t.Run("test add key finalize all sig", testAddKeyFinalize)
}

func testAddKeyForKOResource(t *testing.T) {
	notr := notary.New(logging.NewTestLogger(), notary.NewDefaultConfig())

	kind := types.NodeSignatureKind_ASSET_NEW
	resID := "resid"
	key := []byte(string("123456"))
	sig := []byte(string("123456"))

	// first try to add a key for invalid resource
	_, ok, err := notr.AddSig(resID, kind, key, sig)
	assert.EqualError(t, err, notary.ErrUnknownResourceID.Error())
	assert.False(t, ok)

	// then try to start twice an aggregate
	err = notr.StartAggregate(resID, kind)
	assert.NoError(t, err)
	err = notr.StartAggregate(resID, kind)
	assert.EqualError(t, err, notary.ErrAggregateSigAlreadyStartedForResource.Error())
}

func testAddKeyForOKResource(t *testing.T) {
	notr := notary.New(logging.NewTestLogger(), notary.NewDefaultConfig())

	kind := types.NodeSignatureKind_ASSET_NEW
	resID := "resid"
	key := []byte(string("123456"))
	sig := []byte(string("123456"))

	err := notr.StartAggregate(resID, kind)
	assert.NoError(t, err)

	// first try to add a key for invalid resource
	_, ok, err := notr.AddSig(resID, kind, key, sig)
	assert.NoError(t, err, notary.ErrUnknownResourceID.Error())
	assert.False(t, ok)
}

func testAddKeyFinalize(t *testing.T) {
	notr := notary.New(logging.NewTestLogger(), notary.NewDefaultConfig())

	kind := types.NodeSignatureKind_ASSET_NEW
	resID := "resid"
	key := []byte(string("123456"))
	sig := []byte(string("123456"))

	// add a valid node
	notr.AddNodePubKey(key)

	err := notr.StartAggregate(resID, kind)
	assert.NoError(t, err)

	// first try to add a key for invalid resource
	sigs, ok, err := notr.AddSig(resID, kind, key, sig)
	assert.NoError(t, err, notary.ErrUnknownResourceID.Error())
	assert.True(t, ok)
	assert.Len(t, sigs, 1)
}
