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

package epochtime_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestEpochSnapshotFunctionallyAfterReload(t *testing.T) {
	now := time.Unix(0, 0).UTC()

	ctx := context.Background()
	service := getEpochServiceMT(t)
	defer service.ctrl.Finish()

	service.broker.EXPECT().Send(gomock.Any()).Times(3)
	// Force creation of first epoch to trigger a snapshot of the first epoch
	service.cb(ctx, now)
	// Force creation of first epoch to trigger a snapshot of the first epoch

	data, _, err := service.GetState("all")
	require.Nil(t, err)

	snapService := getEpochServiceMT(t)
	defer snapService.ctrl.Finish()

	snapService.broker.EXPECT().Send(gomock.Any()).Times(2)
	// Fiddle it into a payload by hand
	snap := &snapshot.Payload{}
	err = proto.Unmarshal(data, snap)
	require.Nil(t, err)

	service.NotifyOnEpoch(onEpoch, onEpochRestore)
	snapService.NotifyOnEpoch(onEpoch, onEpochRestore)

	_, err = snapService.LoadState(
		ctx,
		types.PayloadFromProto(snap),
	)
	require.Nil(t, err)

	// Check functional equivalence by stepping forward in time/blocks
	// Reset global used in callback so that is doesn't pick up state from another test
	epochs = []types.Epoch{}

	// Move time forward in time a small amount that should cause no change
	nt := now.Add(time.Hour)
	service.cb(ctx, nt)
	snapService.cb(ctx, nt)
	require.Equal(t, 0, len(epochs))

	// Now send end block
	service.OnBlockEnd(ctx)
	snapService.OnBlockEnd((ctx))
	service.cb(ctx, nt)
	snapService.cb(ctx, nt)
	require.Equal(t, 0, len(epochs))

	// Move even further forward
	nt = now.Add(time.Hour * 25)
	service.cb(ctx, nt)
	snapService.cb(ctx, nt)
	service.OnBlockEnd(ctx)
	snapService.OnBlockEnd((ctx))
	nt = now.Add(time.Hour * 50)
	service.cb(ctx, nt)
	snapService.cb(ctx, nt)
	require.Equal(t, 4, len(epochs))

	// epochs = {start, end, start, end}
	require.Equal(t, epochs[0], epochs[2])
	require.Equal(t, epochs[1], epochs[3])
}

func TestEpochSnapshotHash(t *testing.T) {
	now := time.Unix(0, 0).UTC()

	ctx := context.Background()
	service := getEpochServiceMT(t)
	defer service.ctrl.Finish()

	service.broker.EXPECT().Send(gomock.Any()).Times(3)
	// Trigger initial block
	service.cb(ctx, now)
	s, _, err := service.GetState("all")
	require.Nil(t, err)
	require.Equal(t, "41a9839f4dc60ac14461f58658c0e1bf7542bd54cbd635f3c0402bef2f07f60f", hex.EncodeToString(crypto.Hash(s)))

	// Shuffle time along
	now = now.Add(25 * time.Hour)
	service.cb(ctx, now)
	service.OnBlockEnd(ctx)
	s, _, err = service.GetState("all")
	require.Nil(t, err)
	require.Equal(t, "074677210f20ebb3427064339ebbd46dbfd5d2381bcd3b3fd126bbdcb05b6697", hex.EncodeToString(crypto.Hash(s)))

	// Shuffle time a bit more
	now = now.Add(25 * time.Hour)
	service.cb(ctx, now)
	s, _, err = service.GetState("all")
	require.Nil(t, err)
	require.Equal(t, "2fb572edea4af9154edeff680e23689ed076d08934c60f8a4c1f5743a614954e", hex.EncodeToString(crypto.Hash(s)))
}

func TestEpochSnapshotCompare(t *testing.T) {
	now := time.Unix(0, 0).UTC()

	ctx := context.Background()
	service := getEpochServiceMT(t)
	defer service.ctrl.Finish()

	service.broker.EXPECT().Send(gomock.Any()).Times(1)

	// Force creation of first epoch to trigger a snapshot of the first epoch
	service.cb(ctx, now)

	data, _, err := service.GetState("all")
	require.Nil(t, err)

	snapService := getEpochServiceMT(t)
	defer snapService.ctrl.Finish()

	// Fiddle it into a payload by hand
	snap := &snapshot.Payload{}
	err = proto.Unmarshal(data, snap)
	require.Nil(t, err)

	_, err = snapService.LoadState(
		ctx,
		types.PayloadFromProto(snap),
	)
	require.Nil(t, err)

	// Check that the snapshot of the snapshot is the same as the original snapshot
	newData, _, err := service.GetState("all")
	require.Nil(t, err)
	require.Equal(t, data, newData)
}

func TestEpochSnapshotAfterCheckpoint(t *testing.T) {
	ctx := context.Background()
	service := getEpochServiceMT(t)
	defer service.ctrl.Finish()

	service.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// load from a checkpoint
	pb := &eventspb.EpochEvent{
		Seq:        10,
		Action:     vega.EpochAction_EPOCH_ACTION_START,
		ExpireTime: 1664369813556378344,
		EndTime:    1664362613556378344,
	}

	cpt, err := proto.Marshal(pb)
	require.NoError(t, err)
	require.NoError(t, service.Load(ctx, cpt))

	// Force creation of first epoch to trigger a snapshot of the first epoch
	service.cb(ctx, time.Unix(0, 1664364245193844630))

	d, _, err := service.GetState("all")
	require.NoError(t, err)
	snap := &snapshot.Payload{}
	err = proto.Unmarshal(d, snap)
	require.Nil(t, err)

	require.NotNil(t, snap.GetEpoch())
	require.Equal(t, uint64(10), snap.GetEpoch().Seq)
}
