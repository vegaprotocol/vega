package epochtime_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/epochtime"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"github.com/stretchr/testify/assert"
)

var (
	epochs []types.Epoch
)

type FakeTime struct {
	listeners []func(context.Context, time.Time)
}

func (ft *FakeTime) NotifyOnTick(f func(context.Context, time.Time)) {
	ft.listeners = append(ft.listeners, f)
}

func (ft *FakeTime) SetTimeNow(ctx context.Context, t time.Time) {
	for _, f := range ft.listeners {
		f(ctx, t)
	}
}

func getEpochService(t *testing.T, vt epochtime.VegaTime) *epochtime.Svc {
	ctx := context.Background()
	log := logging.NewTestLogger()
	broker, err := broker.New(ctx, log, broker.NewDefaultConfig())
	assert.NoError(t, err)

	et := epochtime.NewService(
		log,
		epochtime.NewDefaultConfig(),
		vt,
		broker,
	)
	_ = et.OnEpochLengthUpdate(ctx, time.Hour*24) // set default epoch duration
	return et
}

func onEpoch(ctx context.Context, e types.Epoch) {
	epochs = append(epochs, e)
}

func TestEpochService(t *testing.T) {
	now := time.Unix(0, 0).UTC()

	ctx := context.Background()
	ft := FakeTime{}
	es := getEpochService(t, &ft)
	assert.NotNil(t, es)

	// Subscribe to epoch updates
	// Reset global used in callback so that is doesn't pick up state from another test
	epochs = []types.Epoch{}
	es.NotifyOnEpoch(onEpoch)

	// Move time forward to generate first epoch
	ft.SetTimeNow(ctx, now)
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
	ft.SetTimeNow(ctx, now)

	// end the block to mark the end of the epoch
	es.OnBlockEnd(ctx)

	// start the next block to start the second epoch
	ft.SetTimeNow(ctx, now)

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
