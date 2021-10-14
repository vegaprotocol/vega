package notary_test

import (
	"bytes"
	"context"
	"testing"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

var allKey = (&types.PayloadNotary{}).Key()

func TestNotarySnapshotEmpty(t *testing.T) {
	notr := getTestNotary(t)
	h1, err := notr.GetHash(allKey)
	require.Nil(t, err)
	require.Equal(t, 32, len(h1))
}

func TestNotarySnapshotVotesKinds(t *testing.T) {
	notr := getTestNotary(t)

	h1, err := notr.GetHash(allKey)
	require.Nil(t, err)

	resID := "resid"
	notr.top.EXPECT().Len().AnyTimes().Return(1)
	notr.top.EXPECT().IsValidatorVegaPubKey(gomock.Any()).AnyTimes().Return(true)

	notr.StartAggregate(resID, types.NodeSignatureKindAssetNew)

	h2, err := notr.GetHash(allKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h2))
}

func populateNotary(t *testing.T, notr *testNotary) {
	t.Helper()
	// First ID/Kind
	resID := "resid1"
	notr.StartAggregate(resID, types.NodeSignatureKindAssetNew)
	notr.StartAggregate(resID, types.NodeSignatureKindAssetWithdrawal)

	key := "123456"
	ns := commandspb.NodeSignature{
		Sig:  []byte("123456"),
		Id:   resID,
		Kind: types.NodeSignatureKindAssetNew,
	}
	_, ok, err := notr.AddSig(context.Background(), key, ns)
	require.True(t, ok)
	require.Nil(t, err)

	// same key different sig
	ns = commandspb.NodeSignature{
		Sig:  []byte("56789"),
		Id:   resID,
		Kind: types.NodeSignatureKindAssetNew,
	}
	_, ok, err = notr.AddSig(context.Background(), key, ns)
	require.True(t, ok)
	require.Nil(t, err)

	// Add another ID/Kind
	resID = "resid2"
	notr.StartAggregate(resID, types.NodeSignatureKindAssetNew)

	ns = commandspb.NodeSignature{
		Sig:  []byte("123456"),
		Id:   resID,
		Kind: types.NodeSignatureKindAssetNew,
	}
	_, ok, err = notr.AddSig(context.Background(), "123456", ns)
	require.True(t, ok)
	require.Nil(t, err)

	// same sig different key (unlikely)
	ns = commandspb.NodeSignature{
		Sig:  []byte("123456"),
		Id:   resID,
		Kind: types.NodeSignatureKindAssetNew,
	}

	_, ok, err = notr.AddSig(context.Background(), "56789", ns)
	require.True(t, ok)
	require.Nil(t, err)
}

func TestNotarySnapshotRoundTrip(t *testing.T) {
	notr := getTestNotary(t)

	notr.top.EXPECT().Len().AnyTimes().Return(1)
	notr.top.EXPECT().IsValidatorVegaPubKey(gomock.Any()).AnyTimes().Return(true)

	populateNotary(t, notr)

	state, err := notr.GetState(allKey)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	snapNotr := getTestNotary(t)
	snapNotr.top.EXPECT().Len().AnyTimes().Return(1)
	snapNotr.top.EXPECT().IsValidatorVegaPubKey(gomock.Any()).AnyTimes().Return(true)

	err = snapNotr.LoadState(types.PayloadFromProto(snap))
	require.Nil(t, err)

	h1, err := notr.GetHash(allKey)
	require.Nil(t, err)
	h2, err := notr.GetHash(allKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(h1, h2))

	// Check the the loaded in (and original) node signatures exist and are perceived to be ok
	_, ok1 := snapNotr.IsSigned(context.Background(), "resid1", types.NodeSignatureKindAssetNew)
	_, ok2 := notr.IsSigned(context.Background(), "resid1", types.NodeSignatureKindAssetNew)
	require.True(t, ok1)
	require.True(t, ok2)
}
