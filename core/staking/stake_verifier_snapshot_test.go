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

package staking_test

import (
	"bytes"
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/staking"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

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

	sv.tsvc.EXPECT().GetTimeNow().Times(1)
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

	snapSV.evtSrc.EXPECT().UpdateStakingStartingBlock(uint64(42)).Times(1)
	snapSV.OnStateLoaded(ctx)
}

func TestSVSnapshotRemoved(t *testing.T) {
	key := removedKey
	ctx := context.Background()
	sv := getStakeVerifierTest(t)
	defer sv.ctrl.Finish()

	sv.tsvc.EXPECT().GetTimeNow().Times(1)
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
