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

package erc20multisig_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestERC20TopologySnapshotEmpty(t *testing.T) {
	top := getTestTopology(t)
	defer top.ctrl.Finish()

	top.OnTick(context.Background(), time.Unix(10, 0))
	// first set the threshold and 1 validator

	// Let's create threshold
	// first assert we have no threshold
	assert.Equal(t, top.GetThreshold(), uint32(0))

	stateVerified, _, err := top.GetState((&types.PayloadERC20MultiSigTopologyVerified{}).Key())
	assert.NoError(t, err)
	assert.NotNil(t, stateVerified)

	snap := &snapshotpb.Payload{}
	err = proto.Unmarshal(stateVerified, snap)
	require.Nil(t, err)

	snapTop := getTestTopology(t)
	defer snapTop.ctrl.Finish()

	snapTop.LoadState(context.Background(), types.PayloadFromProto(snap))
	state2, _, err := snapTop.GetState((&types.PayloadERC20MultiSigTopologyVerified{}).Key())
	assert.NoError(t, err)
	assert.NotNil(t, state2)
	assert.True(t, bytes.Equal(stateVerified, state2))
}

func TestERC20TopologySnapshot(t *testing.T) {
	top := getTestTopology(t)
	defer top.ctrl.Finish()

	top.OnTick(context.Background(), time.Unix(10, 0))
	// first set the threshold and 1 validator

	// Let's create threshold
	// first assert we have no threshold
	assert.Equal(t, top.GetThreshold(), uint32(0))

	thresholdEvent1 := types.SignerThresholdSetEvent{
		Threshold:   666,
		BlockNumber: 10,
		LogIndex:    11,
		TxHash:      "0xacbde",
		ID:          "someid",
		Nonce:       "123",
		BlockTime:   123456789,
	}

	var cb func(interface{}, bool)
	var res validators.Resource
	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		cb = f
		res = r
		return nil
	})

	assert.NoError(t, top.ProcessThresholdEvent(&thresholdEvent1))

	// now we can call the callback
	cb(res, true)

	// now we can update the time
	top.broker.EXPECT().Send(gomock.Any()).Times(1)
	top.OnTick(context.Background(), time.Unix(11, 0))
	assert.Equal(t, top.GetThreshold(), uint32(666))

	// now the signer

	// first assert we have no signers
	assert.Len(t, top.GetSigners(), 0)

	signerEvent1 := types.SignerEvent{
		BlockNumber: 10,
		LogIndex:    11,
		TxHash:      "0xacbde",
		ID:          "someid",
		Address:     "0xe3133A829FB11c3ad86A992D6576ec7705B105e5",
		Nonce:       "123",
		BlockTime:   123456789,
		Kind:        types.SignerEventKindAdded,
	}

	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		cb = f
		res = r
		return nil
	})

	assert.NoError(t, top.ProcessSignerEvent(&signerEvent1))

	// now we can call the callback
	cb(res, true)

	// now we can update the time
	top.broker.EXPECT().Send(gomock.Any()).Times(1)
	top.OnTick(context.Background(), time.Unix(12, 0))

	t.Run("ensure the signer list is updated", func(t *testing.T) {
		signers := top.GetSigners()
		assert.Len(t, signers, 1)
		assert.Equal(t, "0xe3133A829FB11c3ad86A992D6576ec7705B105e5", signers[0])
	})

	t.Run("check if our party IsSigner", func(t *testing.T) {
		assert.True(t, top.IsSigner("0xe3133A829FB11c3ad86A992D6576ec7705B105e5"))
	})

	t.Run("check excess signers", func(t *testing.T) {
		okAddresses := []string{"0xe3133A829FB11c3ad86A992D6576ec7705B105e5"}
		koAddresses := []string{}

		assert.True(t, top.ExcessSigners(koAddresses))
		assert.False(t, top.ExcessSigners(okAddresses))
	})

	// now we will add some pending ones

	thresholdEvent2 := types.SignerThresholdSetEvent{
		Threshold:   500,
		BlockNumber: 100,
		LogIndex:    1,
		TxHash:      "0xacbde2",
		ID:          "someidthreshold2",
		Nonce:       "1234",
		BlockTime:   123456790,
	}

	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		return nil
	})

	assert.NoError(t, top.ProcessThresholdEvent(&thresholdEvent2))

	signerEvent2 := types.SignerEvent{
		BlockNumber: 101,
		LogIndex:    19,
		TxHash:      "0xacbde3",
		ID:          "someid3",
		Address:     "0xe82EfC4187705655C9b484dFFA25f240e8A6B0BA",
		Nonce:       "1239",
		BlockTime:   123456800,
		Kind:        types.SignerEventKindAdded,
	}

	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		return nil
	})

	assert.NoError(t, top.ProcessSignerEvent(&signerEvent2))

	// now we can snapshot
	stateVerified, _, err := top.GetState((&types.PayloadERC20MultiSigTopologyVerified{}).Key())
	assert.NoError(t, err)
	assert.NotNil(t, stateVerified)
	statePending, _, err := top.GetState((&types.PayloadERC20MultiSigTopologyPending{}).Key())
	assert.NoError(t, err)
	assert.NotNil(t, statePending)

	// now instantiate a new one, and load the stuff
	top2 := getTestTopology(t)
	defer top2.ctrl.Finish()

	snap := &snapshotpb.Payload{}
	err = proto.Unmarshal(stateVerified, snap)
	require.NoError(t, err)

	_, err = top2.LoadState(context.Background(), types.PayloadFromProto(snap))
	assert.NoError(t, err)

	ress := []validators.Resource{}
	cbs := []func(interface{}, bool){}
	// we should have 2 resources being restored
	top2.witness.EXPECT().RestoreResource(gomock.Any(), gomock.Any()).Times(2).DoAndReturn(
		func(res validators.Resource, f func(interface{}, bool)) error {
			ress = append(ress, res)
			cbs = append(cbs, f)
			return nil
		})

	snap2 := &snapshotpb.Payload{}
	err = proto.Unmarshal(statePending, snap2)
	require.NoError(t, err)

	_, err = top2.LoadState(context.Background(), types.PayloadFromProto(snap2))
	assert.NoError(t, err)

	// we should have had 2 callbacks
	assert.Len(t, ress, 2)
	assert.Len(t, cbs, 2)

	// for now we still should have 2 pending, and 2 non pending
	// we can compare states, they should be the same
	tStateVerified, _, err := top.GetState((&types.PayloadERC20MultiSigTopologyVerified{}).Key())
	assert.NoError(t, err)
	assert.Equal(t,
		hex.EncodeToString(crypto.Hash(tStateVerified)),
		"4c7e26567d478632d84de6f68a448a1fe2ea1c20c0b61bca9401cd684b662f32",
	)
	tStatePending, _, err := top.GetState((&types.PayloadERC20MultiSigTopologyPending{}).Key())
	assert.NoError(t, err)
	assert.Equal(t,
		hex.EncodeToString(crypto.Hash(tStatePending)),
		"999cb1f713fb868d83c7bf83d8167a8ef1a1ac8101d68db03307eda02783c71a",
	)

	t2StateVerified, _, err := top2.GetState((&types.PayloadERC20MultiSigTopologyVerified{}).Key())
	assert.NoError(t, err)
	assert.Equal(t,
		hex.EncodeToString(crypto.Hash(t2StateVerified)),
		"4c7e26567d478632d84de6f68a448a1fe2ea1c20c0b61bca9401cd684b662f32",
	)
	t2StatePending, _, err := top2.GetState((&types.PayloadERC20MultiSigTopologyPending{}).Key())
	assert.NoError(t, err)
	assert.Equal(t,
		hex.EncodeToString(crypto.Hash(t2StatePending)),
		"999cb1f713fb868d83c7bf83d8167a8ef1a1ac8101d68db03307eda02783c71a",
	)

	assert.Equal(t, top2.GetThreshold(), uint32(666))
	signers2 := top2.GetSigners()
	assert.Equal(t, signers2[0], "0xe3133A829FB11c3ad86A992D6576ec7705B105e5")
	assert.Len(t, signers2, 1)

	// now let's call the callbacks, and move time
	cbs[0](ress[0], true)
	cbs[1](ress[1], true)

	top2.broker.EXPECT().Send(gomock.Any()).Times(2)
	top2.OnTick(context.Background(), time.Unix(20, 0))

	// now we assert the changes
	assert.Equal(t, top2.GetThreshold(), uint32(500))
	signers3 := top2.GetSigners()
	assert.Equal(t, signers3[0], "0xe3133A829FB11c3ad86A992D6576ec7705B105e5")
	assert.Equal(t, signers3[1], "0xe82EfC4187705655C9b484dFFA25f240e8A6B0BA")
	assert.Len(t, signers3, 2)

	// now let's just check the hash
	t2StateVerifiedLast, _, err := top2.GetState((&types.PayloadERC20MultiSigTopologyVerified{}).Key())
	assert.NoError(t, err)
	assert.Equal(t,
		hex.EncodeToString(crypto.Hash(t2StateVerifiedLast)),
		"bfc99f0b9cfb928f8fc70d162fb2adcb7a064c78e89acfbe559665c17a1311f5",
	)
	t2StatePendingLast, _, err := top2.GetState((&types.PayloadERC20MultiSigTopologyPending{}).Key())
	assert.NoError(t, err)
	assert.Equal(t,
		hex.EncodeToString(crypto.Hash(t2StatePendingLast)),
		"d83b503ec2f409ef3eef86b972d624f750e05e7504c9bdc647e82c46096671fd",
	)
}

func TestERC20TopologySnapshotAddRemoveSigner(t *testing.T) {
	top := getTestTopology(t)
	defer top.ctrl.Finish()

	top.OnTick(context.Background(), time.Unix(10, 0))

	var cb func(interface{}, bool)
	var res validators.Resource
	// first assert we have no signers
	assert.Len(t, top.GetSigners(), 0)

	signerEvent1 := types.SignerEvent{
		BlockNumber: 10,
		LogIndex:    11,
		TxHash:      "0xacbde",
		ID:          "someid",
		Address:     "0xe3133A829FB11c3ad86A992D6576ec7705B105e5",
		Nonce:       "123",
		BlockTime:   123456789,
		Kind:        types.SignerEventKindAdded,
	}

	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		cb = f
		res = r
		return nil
	})

	assert.NoError(t, top.ProcessSignerEvent(&signerEvent1))

	// now we can call the callback
	cb(res, true)

	// now we can update the time
	top.broker.EXPECT().Send(gomock.Any()).Times(1)
	top.OnTick(context.Background(), time.Unix(12, 0))

	// Now we have a signer
	t.Run("ensure the signer list is updated", func(t *testing.T) {
		signers := top.GetSigners()
		assert.Len(t, signers, 1)
		assert.Equal(t, "0xe3133A829FB11c3ad86A992D6576ec7705B105e5", signers[0])
	})

	signerEvent2 := types.SignerEvent{
		BlockNumber: 11,
		LogIndex:    12,
		TxHash:      "0xacbde",
		ID:          "someid",
		Address:     "0xe3133A829FB11c3ad86A992D6576ec7705B105e5",
		Nonce:       "123",
		BlockTime:   123456789,
		Kind:        types.SignerEventKindRemoved,
	}

	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		return nil
	})

	assert.NoError(t, top.ProcessSignerEvent(&signerEvent2))

	// now we can call the callback
	cb(res, true)

	// now we can update the time
	top.broker.EXPECT().Send(gomock.Any()).Times(1)
	top.OnTick(context.Background(), time.Unix(15, 0))

	// Now we have no signer, but some seen events
	t.Run("ensure the signer has been removed", func(t *testing.T) {
		signers := top.GetSigners()
		require.Len(t, signers, 0)
	})

	// now we can snapshot
	stateVerified, _, err := top.GetState((&types.PayloadERC20MultiSigTopologyVerified{}).Key())
	assert.NoError(t, err)
	assert.NotNil(t, stateVerified)

	// now instantiate a new one, and load the stuff
	top2 := getTestTopology(t)
	defer top2.ctrl.Finish()

	snap := &snapshotpb.Payload{}
	err = proto.Unmarshal(stateVerified, snap)
	require.NoError(t, err)

	_, err = top2.LoadState(context.Background(), types.PayloadFromProto(snap))
	assert.NoError(t, err)

	// no signers because they were all removed
	signers2 := top2.GetSigners()
	assert.Len(t, signers2, 0)

	// take a checkpoint to be sure that addressesPerEvents were restored properly
	b1, err := top.Checkpoint()
	require.NoError(t, err)

	b2, err := top2.Checkpoint()
	require.NoError(t, err)

	require.Equal(t, b1, b2)
}
