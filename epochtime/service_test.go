package epochtime_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/epochtime"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/stretchr/testify/assert"
)

var (
	vt     *vegatime.Svc
	now    time.Time = time.Unix(0, 0).UTC()
	epochs []types.Epoch
)

func getEpochService(t *testing.T) *epochtime.Svc {
	ctx := context.Background()
	vt = vegatime.New(vegatime.NewDefaultConfig())
	broker := broker.New(ctx)
	log := logging.NewTestLogger()
	np := netparams.New(log, netparams.NewDefaultConfig(), broker)

	et := epochtime.NewService(
		log,
		epochtime.NewDefaultConfig(),
		vt,
		np,
		broker,
	)
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

	// Move time forward one day
	now = now.Add((time.Hour * 24) + time.Second)
	vt.SetTimeNow(ctx, now)
	// We should have 1 new updates, one for end of epoch.
	assert.Equal(t, 2, len(epochs))
	epoch = epochs[1]
	assert.EqualValues(t, 0, epoch.Seq)
	assert.Equal(t, now.String(), epoch.EndTime.String())

	// Move time forward one block
	now = now.Add(time.Second)
	vt.SetTimeNow(ctx, now)
	// One update for the new epoch
	assert.Equal(t, 3, len(epochs))
	epoch = epochs[2]
	assert.EqualValues(t, 1, epoch.Seq)
	assert.Equal(t, now.String(), epoch.StartTime.String())
	nextDay = now.Add(time.Hour * 24)
	assert.Equal(t, nextDay.String(), epoch.ExpireTime.String())
	assert.True(t, epoch.EndTime.IsZero())
}
