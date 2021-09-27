package epochtime_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/types"
	"github.com/stretchr/testify/require"
)

func TestEpochSnapshotPairwise(t *testing.T) {

	ctx := context.Background()
	service := getEpochService(t)

	// Force creation of first epoch to trigger a snapshot of the first epoch
	vt.SetTimeNow(ctx, now)

	snapshot, err := service.Snapshot()
	require.Nil(t, err)
	require.Equal(t, 1, len(snapshot)) //should be one "chunk"

	snapService := getEpochService(t)
	snapService.LoadSnapshot(snapshot)

	// Check that the snapshot of the snapshot is the same as the original snapshot
	newSnapshot, err := snapService.Snapshot()
	require.Nil(t, err)
	require.Equal(t, snapshot, newSnapshot)

	// Check functional equivalence by stepping forward in time/blocks
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

	ctx := context.Background()
	service := getEpochService(t)

	// Trigger initial block
	vt.SetTimeNow(ctx, now)
	h, err := service.GetHash("")
	require.Nil(t, err)
	require.Equal(t, hex.EncodeToString(h), "c7868aa2fc1beb249876668a99878c3c34b87e3ff5b4768d784582a5cd428aa0")

	// Shuffle time along
	vt.SetTimeNow(ctx, now.Add(time.Hour*25))
	h, err = service.GetHash("")
	require.Nil(t, err)
	require.Equal(t, hex.EncodeToString(h), "0bb058c5466345392f998386b340f1b85bc3518c64210e6bf907f7009cda596b")

	// Block ends
	service.OnBlockEnd(ctx)
	h, err = service.GetHash("")
	require.Nil(t, err)
	require.Equal(t, hex.EncodeToString(h), "c7868aa2fc1beb249876668a99878c3c34b87e3ff5b4768d784582a5cd428aa0")

	// Shuffle time a bit more
	vt.SetTimeNow(ctx, now.Add(time.Hour*50))
	h, err = service.GetHash("")
	require.Nil(t, err)
	require.Equal(t, hex.EncodeToString(h), "9882b9cb17fc1a6d4149737996625b6c29040798af16f2efbfcc3e9260d3ac2b")

}

func TestEpochSnapshotCorrupt(t *testing.T) {

	service := getEpochService(t)

	snapshot := map[string][]byte{
		"invalidkey": {0},
	}

	err := service.LoadSnapshot(snapshot)
	require.NotNil(t, err)

	// Add nonsense bytes to correct key
	snapshot["all"] = []byte{0, 1, 0, 3, 3}
	err = service.LoadSnapshot(snapshot)
	require.NotNil(t, err)

}
