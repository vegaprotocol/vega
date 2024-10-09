// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package volumediscount_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/volumediscount"
	"code.vegaprotocol.io/vega/core/volumediscount/mocks"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/require"
)

type vdEvents interface {
	*events.VolumeDiscountStatsUpdated | *events.VolumeDiscountProgramStarted | *events.VolumeDiscountProgramUpdated | *events.VolumeDiscountProgramEnded
}

type eventMatcher[T vdEvents] struct{}

func (_ eventMatcher[T]) Matches(x any) bool {
	_, ok := x.(T)
	return ok
}

func (_ eventMatcher[T]) String() string {
	var e T
	return fmt.Sprintf("matches %T", e)
}

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

	broker.EXPECT().Send(eventMatcher[*events.VolumeDiscountProgramEnded]{}).DoAndReturn(func(evt events.Event) {
		e := evt.(*events.VolumeDiscountProgramEnded)
		require.Equal(t, p1.Version, e.GetVolumeDiscountProgramEnded().Version)
	}).Times(1)
}

func expectStatsUpdated(t *testing.T, broker *mocks.MockBroker) {
	t.Helper()

	broker.EXPECT().Send(eventMatcher[*events.VolumeDiscountStatsUpdated]{}).Do(func(evt events.Event) {
		_, ok := evt.(*events.VolumeDiscountStatsUpdated)
		require.Truef(t, ok, "expecting event of type *events.VolumeDiscountStatsUpdated but got %T", evt)
	}).Times(1)
}

func expectStatsUpdatedWithUnqualifiedParties(t *testing.T, broker *mocks.MockBroker) {
	t.Helper()

	broker.EXPECT().Send(eventMatcher[*events.VolumeDiscountStatsUpdated]{}).Do(func(evt events.Event) {
		update, ok := evt.(*events.VolumeDiscountStatsUpdated)
		require.Truef(t, ok, "expecting event of type *events.VolumeDiscountStatsUpdated but got %T", evt)
		stats := update.VolumeDiscountStatsUpdated()
		foundUnqualifiedParty := false
		for _, s := range stats.Stats {
			if s.PartyId == "p1" {
				foundUnqualifiedParty = true
				require.Equal(t, "", s.DiscountFactor)
				require.Equal(t, "900", s.RunningVolume)
			}
		}
		require.True(t, foundUnqualifiedParty)
	}).Times(1)
}

func expectProgramStarted(t *testing.T, broker *mocks.MockBroker, p1 *types.VolumeDiscountProgram) {
	t.Helper()

	broker.EXPECT().Send(eventMatcher[*events.VolumeDiscountProgramStarted]{}).Do(func(evt events.Event) {
		e, ok := evt.(*events.VolumeDiscountProgramStarted)
		require.Truef(t, ok, "expecting event of type *events.VolumeDiscountProgramStarted but got %T", evt)
		require.Equal(t, p1.IntoProto(), e.GetVolumeDiscountProgramStarted().Program)
	}).Times(1)
}
