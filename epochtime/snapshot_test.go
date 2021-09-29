package epochtime_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func TestEpochSnapshotFunctionallyAfterReload(t *testing.T) {
	now := time.Unix(0, 0).UTC()

	ft := FakeTime{}
	ctx := context.Background()
	service := getEpochService(t, &ft)

	// Force creation of first epoch to trigger a snapshot of the first epoch
	ft.SetTimeNow(ctx, now)

	data, err := service.Snapshot()
	require.Nil(t, err)
	require.Equal(t, 1, len(data)) //should be one "chunk"

	snapService := getEpochService(t, &ft)

	// Fiddle it into a payload by hand
	snap := &snapshot.Payload{}
	err = proto.Unmarshal(data["all"], snap)
	require.Nil(t, err)

	err = snapService.LoadState(
		types.PayloadFromProto(snap),
	)
	require.Nil(t, err)

	// Check functional equivalence by stepping forward in time/blocks
	// Reset global used in callback so that is doesn't pick up state from another test
	epochs = []types.Epoch{}
	service.NotifyOnEpoch(onEpoch)
	snapService.NotifyOnEpoch(onEpoch)

	// Move time forward in time a small amount that should cause no change
	ft.SetTimeNow(ctx, now.Add(time.Hour))
	require.Equal(t, 0, len(epochs))

	// Now send end block
	service.OnBlockEnd(ctx)
	snapService.OnBlockEnd((ctx))
	ft.SetTimeNow(ctx, now.Add(time.Hour))
	require.Equal(t, 0, len(epochs))

	// Move even further forward
	ft.SetTimeNow(ctx, now.Add(time.Hour*25))
	service.OnBlockEnd(ctx)
	snapService.OnBlockEnd((ctx))
	ft.SetTimeNow(ctx, now.Add(time.Hour*50))
	require.Equal(t, 4, len(epochs))

	// epochs = {start, end, start, end}
	require.Equal(t, epochs[0], epochs[2])
	require.Equal(t, epochs[1], epochs[3])

}

func TestEpochSnapshotHash(t *testing.T) {
	now := time.Unix(0, 0).UTC()

	ctx := context.Background()
	ft := FakeTime{}
	service := getEpochService(t, &ft)

	// Trigger initial block
	ft.SetTimeNow(ctx, now)
	h, err := service.GetHash("all")
	require.Nil(t, err)
	require.Equal(t, "010bd3281c2cdc839fdd0a3bdf0877b174c47980e7c4790ba32befd802a9e1e1", hex.EncodeToString(h))

	// Shuffle time along
	ft.SetTimeNow(ctx, now.Add(time.Hour*25))
	service.OnBlockEnd(ctx)
	h, err = service.GetHash("all")
	require.Nil(t, err)
	require.Equal(t, "e4bbd70ef0aaf86065c14baeeda63d4a13d9cc95e75edb0197ba7bb619683611", hex.EncodeToString(h))

	// Shuffle time a bit more
	ft.SetTimeNow(ctx, now.Add(time.Hour*50))
	h, err = service.GetHash("all")
	require.Nil(t, err)
	require.Equal(t, "9b1cddbbd648b44569a22551b1f1e82379b6d6c664b3e01c18d0ef3edb9a197d", hex.EncodeToString(h))

}

func TestEpochSnapshotCompare(t *testing.T) {
	now := time.Unix(0, 0).UTC()

	ctx := context.Background()
	vt := vegatime.New(vegatime.NewDefaultConfig())
	service := getEpochService(t, vt)

	// Force creation of first epoch to trigger a snapshot of the first epoch
	vt.SetTimeNow(ctx, now)

	data, err := service.Snapshot()
	require.Nil(t, err)
	require.Equal(t, 1, len(data)) //should be one "chunk"

	snapService := getEpochService(t, vt)

	// Fiddle it into a payload by hand
	snap := &snapshot.Payload{}
	err = proto.Unmarshal(data["all"], snap)
	require.Nil(t, err)

	err = snapService.LoadState(
		types.PayloadFromProto(snap),
	)
	require.Nil(t, err)

	// Check that the snapshot of the snapshot is the same as the original snapshot
	newSnapshot, err := snapService.Snapshot()
	require.Nil(t, err)
	require.Equal(t, data, newSnapshot)

	h1, err := service.GetHash("all")
	require.Nil(t, err)
	h2, err := snapService.GetHash("all")
	require.Nil(t, err)

	// Compare hashes
	require.Equal(t, h1, h2)
}
