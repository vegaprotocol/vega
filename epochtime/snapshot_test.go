package epochtime_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
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

	_, err = snapService.LoadState(
		ctx,
		types.PayloadFromProto(snap),
	)
	require.Nil(t, err)

	// Check functional equivalence by stepping forward in time/blocks
	// Reset global used in callback so that is doesn't pick up state from another test
	epochs = []types.Epoch{}
	service.NotifyOnEpoch(onEpoch)
	snapService.NotifyOnEpoch(onEpoch)

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
	h, err := service.GetHash("all")
	require.Nil(t, err)
	require.Equal(t, "41a9839f4dc60ac14461f58658c0e1bf7542bd54cbd635f3c0402bef2f07f60f", hex.EncodeToString(h))

	// Shuffle time along
	now = now.Add(25 * time.Hour)
	service.cb(ctx, now)
	service.OnBlockEnd(ctx)
	h, err = service.GetHash("all")
	require.Nil(t, err)
	require.Equal(t, "074677210f20ebb3427064339ebbd46dbfd5d2381bcd3b3fd126bbdcb05b6697", hex.EncodeToString(h))

	// Shuffle time a bit more
	now = now.Add(25 * time.Hour)
	service.cb(ctx, now)
	h, err = service.GetHash("all")
	require.Nil(t, err)
	require.Equal(t, "2fb572edea4af9154edeff680e23689ed076d08934c60f8a4c1f5743a614954e", hex.EncodeToString(h))
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

	h1, err := service.GetHash("all")
	require.Nil(t, err)
	h2, err := snapService.GetHash("all")
	require.Nil(t, err)

	// Compare hashes
	require.Equal(t, h1, h2)
}
