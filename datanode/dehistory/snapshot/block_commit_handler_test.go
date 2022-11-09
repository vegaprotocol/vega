package snapshot_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/datanode/dehistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"

	"github.com/stretchr/testify/assert"
)

func TestAlteringSnapshotIntervalBelowMinIntervalWithFileSource(t *testing.T) {
	log := logging.NewTestLogger()

	snapshotInterval := &struct {
		interval int
	}{
		interval: 1000,
	}

	var snapshots []int64

	snapshotData := func(ctx context.Context, chainID string, toHeight int64) error {
		if toHeight >= 5000 {
			snapshotInterval.interval = 300
		}

		snapshots = append(snapshots, toHeight)
		return nil
	}

	getNetworkParameter := func(ctx context.Context, key string) (entities.NetworkParameter, error) {
		assert.Equal(t, netparams.SnapshotIntervalLength, key)

		return entities.NetworkParameter{
			Key:   netparams.SnapshotIntervalLength,
			Value: strconv.Itoa(snapshotInterval.interval),
		}, nil
	}

	commitHandler := snapshot.NewBlockCommitHandler(log, snapshotData, getNetworkParameter, true, time.Duration(0))

	ctx := context.Background()
	for blockHeight := int64(0); blockHeight < 6100; blockHeight++ {
		commitHandler.OnBlockCommitted(ctx, "", blockHeight)
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

	getNetworkParameter := func(ctx context.Context, key string) (entities.NetworkParameter, error) {
		assert.Equal(t, netparams.SnapshotIntervalLength, key)

		return entities.NetworkParameter{
			Key:   netparams.SnapshotIntervalLength,
			Value: strconv.Itoa(snapshotInterval.interval),
		}, nil
	}

	commitHandler := snapshot.NewBlockCommitHandler(log, snapshotData, getNetworkParameter, false, time.Duration(0))

	ctx := context.Background()
	for blockHeight := int64(0); blockHeight < 6100; blockHeight++ {
		commitHandler.OnBlockCommitted(ctx, "", blockHeight)
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
