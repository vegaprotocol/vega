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

package volumerebate_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/volumerebate"
	"code.vegaprotocol.io/vega/core/volumerebate/mocks"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/require"
)

// event matchers for relevant events.
var (
	endedEvt   = evtMatcher[events.VolumeRebateProgramEnded]{}
	startedEvt = evtMatcher[events.VolumeRebateProgramStarted]{}
	updatedEvt = evtMatcher[events.VolumeRebateProgramUpdated]{}
	statsEvt   = evtMatcher[events.VolumeRebateStatsUpdated]{}
)

type evts interface {
	events.VolumeRebateStatsUpdated | events.VolumeRebateProgramStarted | events.VolumeRebateProgramUpdated | events.VolumeRebateProgramEnded
}

type evtMatcher[T evts] struct{}

func (_ evtMatcher[T]) String() string {
	var e *T
	return fmt.Sprintf("matches %T", e)
}

func (_ evtMatcher[T]) Matches(x any) bool {
	_, ok := x.(*T)
	return ok
}

// cast uses the matcher for the type assertions in the callbacks, returns nil if the input is incompatible, using the correct matcher should make that impossible.
func (_ evtMatcher[T]) cast(v any) *T {
	e, ok := v.(*T)
	if !ok {
		return nil
	}
	return e
}

func endEpoch(t *testing.T, engine *volumerebate.SnapshottedEngine, seq uint64, endTime time.Time) {
	t.Helper()

	engine.OnEpoch(context.Background(), types.Epoch{
		Seq:     seq,
		EndTime: endTime,
		Action:  vegapb.EpochAction_EPOCH_ACTION_END,
	})
}

func startEpoch(t *testing.T, engine *volumerebate.SnapshottedEngine, seq uint64, startTime time.Time) {
	t.Helper()

	engine.OnEpoch(context.Background(), types.Epoch{
		Seq:       seq,
		StartTime: startTime,
		Action:    vegapb.EpochAction_EPOCH_ACTION_START,
	})
}

func expectProgramEnded(t *testing.T, broker *mocks.MockBroker, p1 *types.VolumeRebateProgram) {
	t.Helper()

	broker.EXPECT().Send(endedEvt).DoAndReturn(func(evt events.Event) {
		e := endedEvt.cast(evt)
		require.Equal(t, p1.Version, e.GetVolumeRebateProgramEnded().Version)
	}).Times(1)
}

func expectStatsUpdated(t *testing.T, broker *mocks.MockBroker) {
	t.Helper()

	broker.EXPECT().Send(statsEvt).Do(func(evt events.Event) {
		e := statsEvt.cast(evt)
		require.NotNil(t, e, "expecting non-nil event of type %s but got %T (nil)", statsEvt, evt)
	}).Times(1)
}

func expectStatsUpdatedWithUnqualifiedParties(t *testing.T, broker *mocks.MockBroker) {
	t.Helper()

	broker.EXPECT().Send(statsEvt).Do(func(evt events.Event) {
		update := statsEvt.cast(evt)
		require.NotNil(t, update, "expecting event of type %s but got %T (nil)", statsEvt, evt)
		stats := update.VolumeRebateStatsUpdated()
		foundUnqualifiedParty := false
		for _, s := range stats.Stats {
			if s.PartyId == "p1" {
				foundUnqualifiedParty = true
				require.Equal(t, "0", s.AdditionalRebate)
				require.Equal(t, "900", s.MakerFeesReceived)
			}
		}
		require.True(t, foundUnqualifiedParty)
	}).Times(1)
}

func expectProgramStarted(t *testing.T, broker *mocks.MockBroker, p1 *types.VolumeRebateProgram) {
	t.Helper()

	broker.EXPECT().Send(startedEvt).Do(func(evt events.Event) {
		e := startedEvt.cast(evt)
		require.NotNil(t, e, "expecting event of type %s but got %T (nil)", startedEvt, evt)
		require.Equal(t, p1.IntoProto(), e.GetVolumeRebateProgramStarted().Program)
	}).Times(1)
}
