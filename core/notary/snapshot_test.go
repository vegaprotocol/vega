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

package notary_test

import (
	"bytes"
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var allKey = (&types.PayloadNotary{}).Key()

func TestNotarySnapshotEmpty(t *testing.T) {
	notr := getTestNotary(t)
	s, _, err := notr.GetState(allKey)
	require.Nil(t, err)
	require.NotNil(t, s)
}

func TestNotarySnapshotVotesKinds(t *testing.T) {
	notr := getTestNotary(t)

	s1, _, err := notr.GetState(allKey)
	require.Nil(t, err)

	resID := "resid"
	notr.top.EXPECT().Len().AnyTimes().Return(1)
	notr.top.EXPECT().IsValidatorVegaPubKey(gomock.Any()).AnyTimes().Return(true)
	notr.top.EXPECT().IsValidator().AnyTimes().Return(true)

	notr.StartAggregate(resID, types.NodeSignatureKindAssetNew, []byte("123456"))

	s2, _, err := notr.GetState(allKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))
}

func populateNotary(t *testing.T, notr *testNotary) {
	t.Helper()
	notr.top.EXPECT().IsValidator().AnyTimes().Return(true)
	// First ID/Kind
	resID := "resid1"
	notr.StartAggregate(resID, types.NodeSignatureKindAssetNew, []byte("123456"))
	notr.StartAggregate(
		resID, types.NodeSignatureKindAssetWithdrawal, []byte("56789"))

	key := "123456"
	ns := commandspb.NodeSignature{
		Sig:  []byte("123456"),
		Id:   resID,
		Kind: types.NodeSignatureKindAssetNew,
	}
	err := notr.RegisterSignature(context.Background(), key, ns)
	require.Nil(t, err)

	// same key different sig
	ns = commandspb.NodeSignature{
		Sig:  []byte("56789"),
		Id:   resID,
		Kind: types.NodeSignatureKindAssetNew,
	}
	err = notr.RegisterSignature(context.Background(), key, ns)
	require.Nil(t, err)

	// Add another ID/Kind
	resID = "resid2"
	notr.StartAggregate(resID, types.NodeSignatureKindAssetNew, []byte("a123456"))

	ns = commandspb.NodeSignature{
		Sig:  []byte("a123456"),
		Id:   resID,
		Kind: types.NodeSignatureKindAssetNew,
	}
	err = notr.RegisterSignature(context.Background(), "123456", ns)
	require.Nil(t, err)

	// same sig different key (unlikely)
	ns = commandspb.NodeSignature{
		Sig:  []byte("b123456"),
		Id:   resID,
		Kind: types.NodeSignatureKindAssetNew,
	}

	err = notr.RegisterSignature(context.Background(), "56789", ns)
	require.Nil(t, err)
}

func TestNotarySnapshotRoundTrip(t *testing.T) {
	notr := getTestNotary(t)
	defer notr.ctrl.Finish()

	notr.top.EXPECT().Len().AnyTimes().Return(1)
	notr.top.EXPECT().IsValidatorVegaPubKey(gomock.Any()).AnyTimes().Return(true)
	notr.top.EXPECT().IsTendermintValidator(gomock.Any()).AnyTimes().Return(true)
	notr.top.EXPECT().IsValidator().AnyTimes().Return(true)
	notr.top.EXPECT().SelfVegaPubKey().AnyTimes().Return("123456")

	populateNotary(t, notr)

	state, _, err := notr.GetState(allKey)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	snapNotr := getTestNotary(t)
	defer snapNotr.ctrl.Finish()
	snapNotr.top.EXPECT().Len().AnyTimes().Return(1)
	snapNotr.top.EXPECT().IsValidator().AnyTimes().Return(true)
	snapNotr.top.EXPECT().SelfVegaPubKey().AnyTimes().Return("123456")
	snapNotr.top.EXPECT().IsValidatorVegaPubKey(gomock.Any()).AnyTimes().Return(true)
	snapNotr.top.EXPECT().IsTendermintValidator(gomock.Any()).AnyTimes().Return(true)

	_, err = snapNotr.LoadState(context.Background(), types.PayloadFromProto(snap))
	require.Nil(t, err)

	s1, _, err := notr.GetState(allKey)
	require.Nil(t, err)
	s2, _, err := notr.GetState(allKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(s1, s2))

	// Check the loaded in (and original) node signatures exist and are perceived to be ok
	_, ok1 := snapNotr.IsSigned(context.Background(), "resid1", types.NodeSignatureKindAssetNew)
	_, ok2 := notr.IsSigned(context.Background(), "resid1", types.NodeSignatureKindAssetNew)
	require.True(t, ok1)
	require.True(t, ok2)
}
