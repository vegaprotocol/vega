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
)

var (
	epochs []types.Epoch
)

type tstSvc struct {
	*epochtime.Svc
	ctrl   *gomock.Controller
	time   *mocks.MockVegaTime
	broker *mbroker.MockBroker
	cb     func(context.Context, time.Time)
}

func getEpochServiceMT(t *testing.T) *tstSvc {
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

func TestEpochService(t *testing.T) {
	now := time.Unix(0, 0).UTC()

	ctx := context.Background()
	service := getEpochServiceMT(t)
	defer service.ctrl.Finish()

	service.broker.EXPECT().Send(gomock.Any()).Times(3)

	// Subscribe to epoch updates
	// Reset global used in callback so that is doesn't pick up state from another test
	epochs = []types.Epoch{}
	service.NotifyOnEpoch(onEpoch)

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
