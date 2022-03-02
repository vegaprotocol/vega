package epochtime_test

import (
	"context"
	"testing"
	"time"

	mbroker "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/epochtime"
	"code.vegaprotocol.io/vega/epochtime/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	epochs   []types.Epoch
	restored []types.Epoch
)

type tstSvc struct {
	*epochtime.Svc
	ctrl   *gomock.Controller
	time   *mocks.MockVegaTime
	broker *mbroker.MockBroker
	cb     func(context.Context, time.Time)
}

func getEpochServiceMT(t *testing.T) *tstSvc {
	t.Helper()
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	tm := mocks.NewMockVegaTime(ctrl)
	broker := mbroker.NewMockBroker(ctrl)
	ret := &tstSvc{
		ctrl:   ctrl,
		time:   tm,
		broker: broker,
	}

	tm.EXPECT().NotifyOnTick(gomock.Any()).Times(1).Do(func(cb func(context.Context, time.Time)) {
		ret.cb = cb
	})

	ret.Svc = epochtime.NewService(
		log,
		epochtime.NewDefaultConfig(),
		tm,
		broker,
	)
	_ = ret.OnEpochLengthUpdate(context.Background(), time.Hour*24) // set default epoch duration
	return ret
}

func onEpoch(ctx context.Context, e types.Epoch) {
	epochs = append(epochs, e)
}

func onEpochRestore(ctx context.Context, e types.Epoch) {
	restored = append(epochs, e)
}

func TestEpochService(t *testing.T) {
	now := time.Unix(0, 0).UTC()

	ctx := context.Background()
	service := getEpochServiceMT(t)
	defer service.ctrl.Finish()

	service.broker.EXPECT().Send(gomock.Any()).Times(3)

	// Subscribe to epoch updates
	// Reset global used in callback so that is doesn't pick up state from another test
	epochs = []types.Epoch{}
	service.NotifyOnEpoch(onEpoch, onEpochRestore)

	// Move time forward to generate first epoch
	service.cb(ctx, now)
	// Check we only have one epoch update
	assert.Equal(t, 1, len(epochs))
	epoch := epochs[0]
	// First epoch will have a 0 identifier
	assert.EqualValues(t, 0, epoch.Seq)
	// Start time should be the same as now
	assert.Equal(t, now.String(), epoch.StartTime.String())
	// Expiry time should 1 day later
	nextDay := now.Add(time.Hour * 24)
	assert.Equal(t, nextDay.String(), epoch.ExpireTime.String())
	// End time should not be set
	assert.True(t, epoch.EndTime.IsZero())

	// Move time forward one day + one second to start the first block past the expiry of the first epoch
	now = now.Add((time.Hour * 24) + time.Second)
	service.cb(ctx, now)

	// end the block to mark the end of the epoch
	service.OnBlockEnd(ctx)

	// start the next block to start the second epoch
	service.cb(ctx, now)

	// We should have 2 new updates, one for end of epoch and one for the beginning of the new one
	assert.Equal(t, 3, len(epochs))
	epoch = epochs[1]
	assert.EqualValues(t, 0, epoch.Seq)
	assert.Equal(t, now.String(), epoch.EndTime.String())

	epoch = epochs[2]
	assert.EqualValues(t, 1, epoch.Seq)
	assert.Equal(t, now.String(), epoch.StartTime.String())
	nextDay = now.Add(time.Hour * 24)
	assert.Equal(t, nextDay.String(), epoch.ExpireTime.String())
	assert.True(t, epoch.EndTime.IsZero())
}

// TestEpochServiceCheckpointLoading tests that when an epoch is loaded from checkpoint within the same epoch, the epoch is not prematurely ending right after the load.
func TestEpochServiceCheckpointLoading(t *testing.T) {
	now := time.Unix(0, 0).UTC()

	ctx := context.Background()
	service := getEpochServiceMT(t)
	defer service.ctrl.Finish()

	service.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Move time forward to generate first epoch
	service.cb(ctx, now)

	// move to 13 hours into the epoch
	now = now.Add(time.Hour * 13)
	println(now.String())
	service.cb(ctx, now)

	// take a checkpoint - 11h to go for the epoch
	cp, _ := service.Checkpoint()

	loadService := getEpochServiceMT(t)
	defer loadService.ctrl.Finish()
	loadService.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	loadEpochs := []types.Epoch{}
	onLoadEpoch := func(ctx context.Context, e types.Epoch) {
		loadEpochs = append(loadEpochs, e)
	}
	loadService.NotifyOnEpoch(onLoadEpoch, onEpochRestore)

	// we're loading the checkpoint 4 hours after the time it was taken but we're still within the same epoch for another few good hours
	now = now.Add((time.Hour * 4))
	println(now.String())
	loadService.cb(ctx, now)
	loadService.Load(ctx, cp)
	// after the load we expect an event regardless of what epoch we were in before
	require.Equal(t, 2, len(loadEpochs))

	// run to the expected end of the epoch and verify it's ended
	now = now.Add((time.Hour * 7) + 1*time.Second)
	println(now.String())
	loadService.cb(ctx, now)
	require.Equal(t, 2, len(loadEpochs))

	loadService.OnBlockEnd(ctx)
	// add another second to start a new epoch
	now = now.Add(1 * time.Second)
	loadService.cb(ctx, now)
	require.Equal(t, 4, len(loadEpochs))
	require.Equal(t, now.String(), loadEpochs[2].EndTime.String())
	require.Equal(t, now.String(), loadEpochs[3].StartTime.String())
}

// TestEpochServiceCheckpointFastForward tests that when an epoch is loaded from checkpoint after the epoch should have ended we're fast forwarding through the epochs that were missed.
func TestEpochServiceCheckpointFastForward(t *testing.T) {
	now := time.Unix(0, 0).UTC()

	ctx := context.Background()
	service := getEpochServiceMT(t)
	defer service.ctrl.Finish()

	service.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Move time forward to generate first epoch
	service.cb(ctx, now)

	// move to 13 hours into the epoch
	now = now.Add(time.Hour * 13)
	println(now.String())
	service.cb(ctx, now)

	// take a checkpoint - 11h to go for the epoch
	// this epoch started at midnight and ends at next midnight
	cp, _ := service.Checkpoint()

	loadService := getEpochServiceMT(t)
	defer loadService.ctrl.Finish()
	loadService.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	loadEpochs := []types.Epoch{}
	onLoadEpoch := func(ctx context.Context, e types.Epoch) {
		loadEpochs = append(loadEpochs, e)
	}
	loadService.NotifyOnEpoch(onLoadEpoch, onEpochRestore)

	// we're loading the checkpoint 4 hours after the time it was taken - so we expect the first epoch (1) to have been finished, as well as epoch 2 and 3 and epoch 4 started
	// 72h means:
	// finished epoch 0 at 2/1 midnight + 1 seconds
	// started epoch 1 at 2/1 midnight + 1 seconds
	// finished epoch 1 at 3/1 midnight + 2 seconds
	// started epoch 2 at 3/1 midnight + 2 seconds
	// ended epoch 2 at 4/1 midnight + 3 seconds
	// started epoch 3 at  4/1 midnight + 3 seconds
	// we're at 13h in epoch 3
	now = now.Add(time.Hour * 72)
	loadService.cb(ctx, now)
	loadService.Load(ctx, cp)

	// new block should trigger fast forward
	loadService.cb(ctx, now)
	require.Equal(t, 8, len(loadEpochs))

	// to advance to the first block after the expiry we need advance by for 11h and 4 seconds
	now = now.Add((time.Hour * 11) + 4*time.Second)
	loadService.cb(ctx, now)
	loadService.OnBlockEnd(ctx)

	// add another second to start a new epoch
	now = now.Add(1 * time.Second)
	loadService.cb(ctx, now)
	require.Equal(t, 10, len(loadEpochs))

	require.Equal(t, uint64(0), loadEpochs[2].Seq)
	require.Equal(t, "1970-01-01 00:00:00 +0000 UTC", loadEpochs[2].StartTime.UTC().String())
	require.Equal(t, "1970-01-02 00:00:01 +0000 UTC", loadEpochs[2].EndTime.UTC().String())
	require.Equal(t, uint64(1), loadEpochs[3].Seq)
	require.Equal(t, "1970-01-02 00:00:01 +0000 UTC", loadEpochs[3].StartTime.UTC().String())
	require.Equal(t, "1970-01-03 00:00:01 +0000 UTC", loadEpochs[3].ExpireTime.UTC().String())
	require.Equal(t, uint64(1), loadEpochs[4].Seq)
	require.Equal(t, "1970-01-02 00:00:01 +0000 UTC", loadEpochs[4].StartTime.UTC().String())
	require.Equal(t, "1970-01-03 00:00:02 +0000 UTC", loadEpochs[4].EndTime.UTC().String())
	require.Equal(t, uint64(2), loadEpochs[5].Seq)
	require.Equal(t, "1970-01-03 00:00:02 +0000 UTC", loadEpochs[5].StartTime.UTC().String())
	require.Equal(t, "1970-01-04 00:00:02 +0000 UTC", loadEpochs[5].ExpireTime.UTC().String())
	require.Equal(t, uint64(2), loadEpochs[6].Seq)
	require.Equal(t, "1970-01-03 00:00:02 +0000 UTC", loadEpochs[6].StartTime.UTC().String())
	require.Equal(t, "1970-01-04 00:00:03 +0000 UTC", loadEpochs[6].EndTime.UTC().String())
	require.Equal(t, uint64(3), loadEpochs[7].Seq)
	require.Equal(t, "1970-01-04 00:00:03 +0000 UTC", loadEpochs[7].StartTime.UTC().String())
	require.Equal(t, "1970-01-05 00:00:03 +0000 UTC", loadEpochs[7].ExpireTime.UTC().String())
	require.Equal(t, uint64(3), loadEpochs[8].Seq)
	require.Equal(t, "1970-01-04 00:00:03 +0000 UTC", loadEpochs[8].StartTime.UTC().String())
	require.Equal(t, "1970-01-05 00:00:05 +0000 UTC", loadEpochs[8].EndTime.UTC().String())
	require.Equal(t, uint64(4), loadEpochs[9].Seq)
	require.Equal(t, "1970-01-05 00:00:05 +0000 UTC", loadEpochs[9].StartTime.UTC().String())
}
