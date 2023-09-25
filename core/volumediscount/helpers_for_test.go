package volumediscount_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/volumediscount"
	"code.vegaprotocol.io/vega/core/volumediscount/mocks"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func endEpoch(t *testing.T, engine *volumediscount.SnapshottedEngine, seq uint64, endTime time.Time) {
	t.Helper()

	engine.OnEpoch(context.Background(), types.Epoch{
		Seq:     seq,
		EndTime: endTime,
		Action:  vegapb.EpochAction_EPOCH_ACTION_END,
	})
}

func startEpoch(t *testing.T, engine *volumediscount.SnapshottedEngine, seq uint64, startTime time.Time) {
	t.Helper()

	engine.OnEpoch(context.Background(), types.Epoch{
		Seq:       seq,
		StartTime: startTime,
		Action:    vegapb.EpochAction_EPOCH_ACTION_START,
	})
}

func expectProgramEnded(t *testing.T, broker *mocks.MockBroker, p1 *types.VolumeDiscountProgram) {
	t.Helper()

	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(evt events.Event) {
		e := evt.(*events.VolumeDiscountProgramEnded)
		require.Equal(t, p1.Version, e.GetVolumeDiscountProgramEnded().Version)
	}).Times(1)
}

func expectStatsUpdated(t *testing.T, broker *mocks.MockBroker) {
	t.Helper()

	broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		_, ok := evt.(*events.VolumeDiscountStatsUpdated)
		require.Truef(t, ok, "expecting event of type *events.VolumeDiscountStatsUpdated but got %T", evt)
	}).Times(1)
}

func expectProgramStarted(t *testing.T, broker *mocks.MockBroker, p1 *types.VolumeDiscountProgram) {
	t.Helper()

	broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		e, ok := evt.(*events.VolumeDiscountProgramStarted)
		require.Truef(t, ok, "expecting event of type *events.VolumeDiscountProgramStarted but got %T", evt)
		require.Equal(t, p1.IntoProto(), e.GetVolumeDiscountProgramStarted().Program)
	}).Times(1)
}
