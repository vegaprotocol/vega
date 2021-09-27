package epochtime_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/epochtime"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/stretchr/testify/assert"
)

var (
	vt     *vegatime.Svc = vegatime.New(vegatime.NewDefaultConfig())
	now    time.Time     = time.Unix(0, 0).UTC()
	epochs []types.Epoch
)

func getEpochService(t *testing.T) *epochtime.Svc {
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
	ctx := context.Background()
	es := getEpochService(t)
	assert.NotNil(t, es)

	// Subscribe to epoch updates
	epochs = []types.Epoch{}
	es.NotifyOnEpoch(onEpoch)

	// Move time forward to generate first epoch
	vt.SetTimeNow(ctx, now)
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
	vt.SetTimeNow(ctx, now)

	// end the block to mark the end of the epoch
	es.OnBlockEnd(ctx)

	// start the next block to start the second epoch
	vt.SetTimeNow(ctx, now)

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
