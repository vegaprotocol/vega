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

func TestEpochSnapshotPairwise(t *testing.T) {
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
	snap := &snapshot.EpochState{}
	err = proto.Unmarshal(data["all"], snap)
	require.Nil(t, err)

	snapService.LoadSnapshot(
		types.PayloadEpochFromProto(
			&snapshot.Payload_Epoch{Epoch: snap},
		),
	)

	// Check that the snapshot of the snapshot is the same as the original snapshot
	newSnapshot, err := snapService.Snapshot()
	require.Nil(t, err)
	require.Equal(t, data, newSnapshot)

	// Check functional equivalence by stepping forward in time/blocks
	// Reset global used in callback so that is doesn't pick up state from another test
	epochs = []types.Epoch{}
	service.NotifyOnEpoch(onEpoch)
	snapService.NotifyOnEpoch(onEpoch)

	// Move time forward in time a small amount that should cause no change
	vt.SetTimeNow(ctx, now.Add(time.Hour))
	require.Equal(t, 0, len(epochs))

	// Now send end block
	service.OnBlockEnd(ctx)
	snapService.OnBlockEnd((ctx))
	vt.SetTimeNow(ctx, now.Add(time.Hour))
	require.Equal(t, 0, len(epochs))

	// Move even further forward
	vt.SetTimeNow(ctx, now.Add(time.Hour*25))
	service.OnBlockEnd(ctx)
	snapService.OnBlockEnd((ctx))
	vt.SetTimeNow(ctx, now.Add(time.Hour*50))
	require.Equal(t, 4, len(epochs))

	// epochs = {start, end, start, end}
	require.Equal(t, epochs[0], epochs[2])
	require.Equal(t, epochs[1], epochs[3])

}

func TestEpochSnapshotHash(t *testing.T) {
	now := time.Unix(0, 0).UTC()

	ctx := context.Background()
	vt := vegatime.New(vegatime.NewDefaultConfig())
	service := getEpochService(t, vt)

	// Trigger initial block
	vt.SetTimeNow(ctx, now)
	h, err := service.GetHash("")
	require.Nil(t, err)
	require.Equal(t, hex.EncodeToString(h), "6072379e85f4b60ec80bad60660189ffe1c7a373d449175f6834f4432dad33f4")

	// Shuffle time along
	vt.SetTimeNow(ctx, now.Add(time.Hour*25))
	h, err = service.GetHash("")
	require.Nil(t, err)
	require.Equal(t, hex.EncodeToString(h), "ca7b6c216960333a6b29f388ed30f55ba7b8a849ed83909d892d095fa7651274")

	// Block ends
	service.OnBlockEnd(ctx)
	h, err = service.GetHash("")
	require.Nil(t, err)
	require.Equal(t, hex.EncodeToString(h), "f7a76fd3d432f9db5c460c8f0860bc77830440434f8aad950f8cc6a7881994c0")

	// Shuffle time a bit more
	vt.SetTimeNow(ctx, now.Add(time.Hour*50))
	h, err = service.GetHash("")
	require.Nil(t, err)
	require.Equal(t, hex.EncodeToString(h), "79f02b031cd59fc134a4fdfb895bee309bebbd748743999c3084c43d1cd8bd32")

}
