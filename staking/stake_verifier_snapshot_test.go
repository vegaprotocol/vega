// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package staking_test

import (
	"bytes"
	"context"
	"testing"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	depositedKey = (&types.PayloadStakeVerifierDeposited{}).Key()
	removedKey   = (&types.PayloadStakeVerifierRemoved{}).Key()
)

func TestSVSnapshotEmpty(t *testing.T) {
	sv := getStakeVerifierTest(t)
	defer sv.ctrl.Finish()

	assert.Equal(t, 2, len(sv.Keys()))

	s, _, err := sv.GetState(depositedKey)
	require.Nil(t, err)
	require.NotNil(t, s)

	s, _, err = sv.GetState(removedKey)
	require.Nil(t, err)
	require.NotNil(t, s)
}

func TestSVSnapshotDeposited(t *testing.T) {
	key := depositedKey
	ctx := context.Background()
	sv := getStakeVerifierTest(t)
	defer sv.ctrl.Finish()

	sv.broker.EXPECT().Send(gomock.Any()).Times(1)
	sv.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	s1, _, err := sv.GetState(key)
	require.Nil(t, err)
	require.NotNil(t, s1)

	event := &types.StakeDeposited{
		BlockNumber:     42,
		LogIndex:        1789,
		TxID:            "somehash",
		ID:              "someid",
		VegaPubKey:      "somepubkey",
		EthereumAddress: "0xnothex",
		Amount:          num.NewUint(1000),
		BlockTime:       100000,
	}

	err = sv.ProcessStakeDeposited(ctx, event)
	require.Nil(t, err)

	s2, _, err := sv.GetState(key)
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))

	state, _, err := sv.GetState(key)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	// Restore into new things
	snapSV := getStakeVerifierTest(t)
	defer snapSV.ctrl.Finish()
	snapSV.witness.EXPECT().RestoreResource(gomock.Any(), gomock.Any()).Times(1)

	snapSV.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	_, err = snapSV.LoadState(ctx, types.PayloadFromProto(snap))
	require.Nil(t, err)
	// Check its there by adding it again and checking for duplication error
	require.ErrorIs(t, staking.ErrDuplicatedStakeDepositedEvent, snapSV.ProcessStakeDeposited(ctx, event))
}

func TestSVSnapshotRemoved(t *testing.T) {
	key := removedKey
	ctx := context.Background()
	sv := getStakeVerifierTest(t)
	defer sv.ctrl.Finish()

	sv.broker.EXPECT().Send(gomock.Any()).Times(1)
	sv.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	s1, _, err := sv.GetState(key)
	require.Nil(t, err)
	require.NotNil(t, s1)

	event := &types.StakeRemoved{
		BlockNumber:     42,
		LogIndex:        1789,
		TxID:            "somehash",
		ID:              "someid",
		VegaPubKey:      "somepubkey",
		EthereumAddress: "0xnothex",
		Amount:          num.NewUint(1000),
		BlockTime:       100000,
	}

	err = sv.ProcessStakeRemoved(ctx, event)
	require.Nil(t, err)

	s2, _, err := sv.GetState(key)
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))

	state, _, err := sv.GetState(key)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	// Restore into new things
	snapSV := getStakeVerifierTest(t)
	defer snapSV.ctrl.Finish()
	snapSV.witness.EXPECT().RestoreResource(gomock.Any(), gomock.Any()).Times(1)

	snapSV.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	_, err = snapSV.LoadState(ctx, types.PayloadFromProto(snap))
	require.Nil(t, err)
	// Check its there by adding it again and checking for duplication error
	require.ErrorIs(t, staking.ErrDuplicatedStakeRemovedEvent, snapSV.ProcessStakeRemoved(ctx, event))
}
