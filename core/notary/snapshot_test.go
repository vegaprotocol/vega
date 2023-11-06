// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package notary_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

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

func TestRetryPendingOnly(t *testing.T) {
	notr := getTestNotary(t)
	defer notr.ctrl.Finish()

	notr.top.EXPECT().Len().AnyTimes().Return(1)
	notr.top.EXPECT().IsValidatorVegaPubKey(gomock.Any()).AnyTimes().Return(true)
	notr.top.EXPECT().IsTendermintValidator(gomock.Any()).AnyTimes().Return(true)
	notr.top.EXPECT().IsValidator().AnyTimes().Return(true)
	notr.top.EXPECT().SelfVegaPubKey().AnyTimes().Return("123456")

	// we will start this signature aggregation but not send in a signature
	notr.StartAggregate(
		"resid1", types.NodeSignatureKindERC20MultiSigSignerAdded, []byte("123444"))

	// we will start another one send in signatures from a pretend other validator
	resID := "resid2"
	notr.StartAggregate(resID, types.NodeSignatureKindAssetNew, []byte("123456"))
	key := "other-validator"
	ns := commandspb.NodeSignature{
		Sig:  []byte("123456"),
		Id:   resID,
		Kind: types.NodeSignatureKindAssetNew,
	}
	err := notr.RegisterSignature(context.Background(), key, ns)
	require.Nil(t, err)

	// get the snapshot state and load into a new engine
	state, _, err := notr.GetState(allKey)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	snapNotr := getTestNotary(t)
	defer snapNotr.ctrl.Finish()
	snapNotr.top.EXPECT().Len().AnyTimes().Return(1)
	snapNotr.top.EXPECT().IsValidator().AnyTimes().Return(true)
	snapNotr.top.EXPECT().SelfVegaPubKey().AnyTimes().Return("this-validator")
	snapNotr.top.EXPECT().IsValidatorVegaPubKey(gomock.Any()).AnyTimes().Return(true)
	snapNotr.top.EXPECT().IsTendermintValidator(gomock.Any()).AnyTimes().Return(true)

	_, err = snapNotr.LoadState(context.Background(), types.PayloadFromProto(snap))
	require.Nil(t, err)

	s1, _, err := notr.GetState(allKey)
	require.Nil(t, err)
	s2, _, err := notr.GetState(allKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(s1, s2))

	snapNotr.cmd.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	snapNotr.OnTick(context.Background(), time.Now())
}
