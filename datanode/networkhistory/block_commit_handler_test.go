package networkhistory_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/networkhistory"

	"code.vegaprotocol.io/vega/logging"

	"github.com/stretchr/testify/assert"
)

func TestAlteringSnapshotIntervalBelowMinIntervalWithFileSource(t *testing.T) {
	log := logging.NewTestLogger()

	var snapshots []int64

	snapshotData := func(ctx context.Context, chainID string, toHeight int64) error {
		snapshots = append(snapshots, toHeight)
		return nil
	}

	commitHandler := networkhistory.NewBlockCommitHandler(log, networkhistory.NewDefaultConfig(), snapshotData, true, time.Duration(0))

	ctx := context.Background()
	for blockHeight := int64(0); blockHeight < 6100; blockHeight++ {
		snapshotTaken := blockHeight%1000 == 0
		if blockHeight >= 5000 {
			snapshotTaken = blockHeight%300 == 0
		}
		commitHandler.OnBlockCommitted(ctx, "", blockHeight, snapshotTaken)
	}

	assert.Equal(t, 6, len(snapshots))
	assert.Equal(t, int64(1000), snapshots[0])
	assert.Equal(t, int64(2000), snapshots[1])
	assert.Equal(t, int64(3000), snapshots[2])
	assert.Equal(t, int64(4000), snapshots[3])
	assert.Equal(t, int64(5000), snapshots[4])
	assert.Equal(t, int64(6000), snapshots[5])
}

func TestAlteringBlockCommitHandlerSnapshotInterval(t *testing.T) {
	log := logging.NewTestLogger()

	var snapshots []int64

	snapshotData := func(ctx context.Context, chainID string, toHeight int64) error {
		snapshots = append(snapshots, toHeight)
		return nil
	}
	commitHandler := networkhistory.NewBlockCommitHandler(log, networkhistory.NewDefaultConfig(), snapshotData, false, time.Duration(0))
	ctx := context.Background()

	for blockHeight := int64(0); blockHeight < 6100; blockHeight++ {
		snapshotTaken := blockHeight%1000 == 0
		if blockHeight >= 5000 {
			snapshotTaken = blockHeight%500 == 0
		}

		commitHandler.OnBlockCommitted(ctx, "", blockHeight, snapshotTaken)
	}

	assert.Equal(t, 7, len(snapshots))
	assert.Equal(t, int64(1000), snapshots[0])
	assert.Equal(t, int64(2000), snapshots[1])
	assert.Equal(t, int64(3000), snapshots[2])
	assert.Equal(t, int64(4000), snapshots[3])
	assert.Equal(t, int64(5000), snapshots[4])
	assert.Equal(t, int64(5500), snapshots[5])
	assert.Equal(t, int64(6000), snapshots[6])
}

func TestPublishingOff(t *testing.T) {
	log := logging.NewTestLogger()

	snapshotInterval := &struct {
		interval int
	}{
		interval: 1000,
	}

	var snapshots []int64

	snapshotData := func(ctx context.Context, chainID string, toHeight int64) error {
		if toHeight >= 5000 {
			snapshotInterval.interval = 500
		}

		snapshots = append(snapshots, toHeight)
		return nil
	}

	cfg := networkhistory.NewDefaultConfig()
	cfg.Publish = false
	commitHandler := networkhistory.NewBlockCommitHandler(log, cfg, snapshotData, false, 0)

	ctx := context.Background()
	for blockHeight := int64(0); blockHeight < 6100; blockHeight++ {
		commitHandler.OnBlockCommitted(ctx, "", blockHeight, true) // show that regardless of what the core says about snapshot taken, none is taken here as publish is false.
	}

	assert.Equal(t, 0, len(snapshots))
}
