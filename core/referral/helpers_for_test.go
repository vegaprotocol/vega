// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package referral_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/referral"
	"code.vegaprotocol.io/vega/core/referral/mocks"
	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	typespb "code.vegaprotocol.io/vega/protos/vega"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEngine struct {
	engine *referral.SnapshottedEngine
	broker *mocks.MockBroker
}

func newPartyID(t *testing.T) types.PartyID {
	t.Helper()

	return types.PartyID(vgrand.RandomStr(5))
}

func newSnapshotEngine(t *testing.T, vegaPath paths.Paths, now time.Time, engine *referral.SnapshottedEngine) *snapshot.Engine {
	t.Helper()

	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snapshot.DefaultConfig()

	snapshotEngine, err := snapshot.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)

	snapshotEngine.AddProviders(engine)

	return snapshotEngine
}

func newEngine(t *testing.T) *testEngine {
	t.Helper()

	ctrl := gomock.NewController(t)

	epochEngine := mocks.NewMockEpochEngine(ctrl)
	epochEngine.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any())

	timeSvc := mocks.NewMockTimeService(ctrl)

	broker := mocks.NewMockBroker(ctrl)

	engine := referral.NewSnapshottedEngine(epochEngine, broker, timeSvc)

	return &testEngine{
		engine: engine,
		broker: broker,
	}
}

func nextEpoch(t *testing.T, ctx context.Context, te *testEngine, startEpochTime time.Time) {
	t.Helper()

	te.engine.OnEpoch(ctx, types.Epoch{
		Action:  typespb.EpochAction_EPOCH_ACTION_END,
		EndTime: startEpochTime.Add(-1 * time.Second),
	})
	te.engine.OnEpoch(ctx, types.Epoch{
		Action:    typespb.EpochAction_EPOCH_ACTION_START,
		StartTime: startEpochTime,
	})
}

func expectReferralProgramStartedEvent(t *testing.T, engine *testEngine) {
	t.Helper()

	engine.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		_, ok := evt.(*events.ReferralProgramStarted)
		assert.True(t, ok, "Event should be a ReferralProgramStarted, but is %T", evt)
	}).Times(1)
}

func expectReferralProgramEndedEvent(t *testing.T, engine *testEngine) *gomock.Call {
	t.Helper()

	return engine.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		_, ok := evt.(*events.ReferralProgramEnded)
		assert.True(t, ok, "Event should be a ReferralProgramEnded, but is %T", evt)
	}).Times(1)
}

func expectReferralProgramUpdatedEvent(t *testing.T, engine *testEngine) *gomock.Call {
	t.Helper()

	return engine.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		_, ok := evt.(*events.ReferralProgramUpdated)
		assert.True(t, ok, "Event should be a ReferralProgramUpdated, but is %T", evt)
	}).Times(1)
}
