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
	require.Equal(t, "e17496cd48a1d1cc0e2715e815edebb6c4e981c1a8b2f7af05351f075a823109", hex.EncodeToString(h))

	// Shuffle time along
	vt.SetTimeNow(ctx, now.Add(time.Hour*25))
	h, err = service.GetHash("")
	require.Nil(t, err)
	require.Equal(t, "4a589cef1aac4ea4162301f774909c65371fd26081d4f55fa560f58c6c7c2f29", hex.EncodeToString(h))

	// Block ends
	service.OnBlockEnd(ctx)
	h, err = service.GetHash("")
	require.Nil(t, err)
	require.Equal(t, "b208a3b963b553f318abd9e80c8ce6d0e37a051fe1018e9b82a3016620c0fb21", hex.EncodeToString(h))

	// Shuffle time a bit more
	vt.SetTimeNow(ctx, now.Add(time.Hour*50))
	h, err = service.GetHash("")
	require.Nil(t, err)
	require.Equal(t, "cc94641b5ed8def9ad0ad293da2e12e1b1e4fc9de14c9343271d5de72a8c61f1", hex.EncodeToString(h))

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

	h1, err := service.GetHash("all")
	require.Nil(t, err)
	h2, err := snapService.GetHash("all")
	require.Nil(t, err)

	// Compare hashes
	require.Equal(t, h1, h2)
}
