package validators_test

import (
	"bytes"
	"context"
	"testing"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var topKey = (&types.PayloadTopology{}).Key()

func TestEmptySnapshot(t *testing.T) {
	top := getTestTopology(t)
	defer top.ctrl.Finish()

	h, err := top.GetHash(topKey)
	assert.Nil(t, err)
	assert.NotEmpty(t, h)

	s, p, err := top.GetState(topKey)
	assert.Nil(t, err)
	assert.Empty(t, p)
	assert.NotEmpty(t, s)

	assert.Equal(t, 1, len(top.Keys()))
}

func TestTopologySnapshot(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()

	h1, err := top.GetHash(topKey)
	require.Nil(t, err)

	tmPubKeys := []string{"tm-pubkey-1", "tm-pubkey-2"}
	top.UpdateValidatorSet(tmPubKeys)

	h2, err := top.GetHash(topKey)
	require.Nil(t, err)

	ctx := context.Background()

	nr1 := commandspb.NodeRegistration{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKeys[0],
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
	}
	err = top.AddNodeRegistration(ctx, &nr1)
	assert.NoError(t, err)

	nr2 := commandspb.NodeRegistration{
		Id:              "vega-master-pubkey-2",
		ChainPubKey:     tmPubKeys[1],
		VegaPubKey:      "vega-key-2",
		EthereumAddress: "eth-address-2",
	}
	err = top.AddNodeRegistration(ctx, &nr2)
	assert.NoError(t, err)

	kr1 := &commandspb.KeyRotateSubmission{
		NewPubKeyIndex:    1,
		TargetBlock:       10,
		NewPubKey:         "new-vega-key",
		CurrentPubKeyHash: hashKey(nr1.VegaPubKey),
	}
	err = top.AddKeyRotate(ctx, nr1.Id, 5, kr1)
	assert.NoError(t, err)

	kr2 := &commandspb.KeyRotateSubmission{
		NewPubKeyIndex:    1,
		TargetBlock:       11,
		NewPubKey:         "new-vega-key-2",
		CurrentPubKeyHash: hashKey(nr2.VegaPubKey),
	}
	err = top.AddKeyRotate(ctx, nr2.Id, 5, kr2)
	assert.NoError(t, err)

	top.BeginBlock(ctx, 10)

	// Check the hashes have changed after each state change
	h3, err := top.GetHash(topKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h2))
	require.False(t, bytes.Equal(h2, h3))
	require.False(t, bytes.Equal(h1, h3))

	// Get the state ready to load into a new instance of the engine
	state, _, _ := top.GetState(topKey)
	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	snapTop := getTestTopWithDefaultValidator(t)
	defer snapTop.ctrl.Finish()

	_, err = snapTop.LoadState(context.Background(), types.PayloadFromProto(snap))
	require.Nil(t, err)

	// Check the new reloaded engine is the same as the original
	h4, err := top.GetHash(topKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(h3, h4))
	assert.ElementsMatch(t, top.AllNodeIDs(), snapTop.AllNodeIDs())
	assert.ElementsMatch(t, top.AllVegaPubKeys(), snapTop.AllVegaPubKeys())
	assert.Equal(t, top.IsValidator(), snapTop.IsValidator())
}
