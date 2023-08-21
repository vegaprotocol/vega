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

package teams_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/teams"
	"code.vegaprotocol.io/vega/core/teams/mocks"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/ptr"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	typespb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEngine struct {
	engine      *teams.SnapshottedEngine
	broker      *mocks.MockBroker
	timeService *mocks.MockTimeService
}

func assertEqualTeams(t *testing.T, expected, actual []types.Team) {
	t.Helper()

	teams.SortByTeamID(expected)

	if len(expected) != len(actual) {
		assert.Fail(t, fmt.Sprintf("Expected len of %d but got %d", len(expected), len(actual)))
	}

	for i := 0; i < len(expected); i++ {
		t.Run(fmt.Sprintf("team %d", i), func(tt *testing.T) {
			expectedTeam := expected[i]
			actualTeam := actual[i]
			assert.Equal(tt, expectedTeam.ID, actualTeam.ID)
			assert.Equal(tt, expectedTeam.Name, actualTeam.Name)
			assert.Equal(tt, expectedTeam.TeamURL, actualTeam.TeamURL)
			assert.Equal(tt, expectedTeam.AvatarURL, actualTeam.AvatarURL)
			assert.Equal(tt, expectedTeam.CreatedAt.UnixNano(), actualTeam.CreatedAt.UnixNano())
			assertEqualMembership(tt, expectedTeam.Referrer, actualTeam.Referrer)

			for j := 0; j < len(expectedTeam.Referees); j++ {
				tt.Run(fmt.Sprintf("referee %d", j), func(ttt *testing.T) {
					assertEqualMembership(ttt, expectedTeam.Referees[j], actualTeam.Referees[j])
				})
			}
		})
	}
}

func assertEqualMembership(t *testing.T, expected, actual *types.Membership) {
	t.Helper()

	assert.Equal(t, expected.PartyID, actual.PartyID)
	assert.Equal(t, expected.JoinedAt.UnixNano(), actual.JoinedAt.UnixNano())
	assert.Equal(t, expected.NumberOfEpoch, actual.NumberOfEpoch)
}

func expectTeamCreatedEvent(t *testing.T, engine *testEngine) {
	t.Helper()

	engine.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		_, ok := evt.(*events.TeamCreated)
		assert.True(t, ok, "Event should be a TeamCreated, but is %T", evt)
	}).Times(1)
}

func expectTeamUpdatedEvent(t *testing.T, engine *testEngine) {
	t.Helper()

	engine.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		_, ok := evt.(*events.TeamUpdated)
		assert.True(t, ok, "Event should be a TeamUpdated, but is %T", evt)
	}).Times(1)
}

func expectRefereeJoinedTeamEvent(t *testing.T, engine *testEngine) {
	t.Helper()

	engine.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		_, ok := evt.(*events.RefereeJoinedTeam)
		assert.True(t, ok, "Event should be a RefereeJoinedTeam, but is %T", evt)
	}).Times(1)
}

func expectRefereeSwitchedTeamEvent(t *testing.T, engine *testEngine) {
	t.Helper()

	engine.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		_, ok := evt.(*events.RefereeSwitchedTeam)
		assert.True(t, ok, "Event should be a RefereeSwitchedTeam, but is %T", evt)
	}).Times(1)
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

func newSnapshotEngine(t *testing.T, vegaPath paths.Paths, now time.Time, engine *teams.SnapshottedEngine) *snapshot.Engine {
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

	broker := mocks.NewMockBroker(ctrl)
	timeService := mocks.NewMockTimeService(ctrl)

	engine := teams.NewSnapshottedEngine(epochEngine, broker, timeService)

	return &testEngine{
		engine:      engine,
		broker:      broker,
		timeService: timeService,
	}
}

func newTeamID(t *testing.T) types.TeamID {
	t.Helper()

	return types.TeamID(vgcrypto.RandomHash())
}

func newPartyID(t *testing.T) types.PartyID {
	t.Helper()

	return types.PartyID(vgrand.RandomStr(5))
}

func newTeam(t *testing.T, ctx context.Context, te *testEngine) (types.TeamID, types.PartyID) {
	t.Helper()

	teamID := newTeamID(t)
	referrer := newPartyID(t)

	expectTeamCreatedEvent(t, te)

	err := te.engine.CreateTeam(ctx, referrer, teamID, createTeamCmd(t, "", "", ""))
	require.NoError(t, err)
	require.NotEmpty(t, teamID)
	require.True(t, te.engine.IsTeamMember(referrer))

	return teamID, referrer
}

func createTeamCmd(t *testing.T, name, teamURL, avatarURL string) *commandspb.CreateTeam {
	t.Helper()

	return &commandspb.CreateTeam{
		Name:      ptr.From(name),
		TeamUrl:   ptr.From(teamURL),
		AvatarUrl: ptr.From(avatarURL),
	}
}

func updateTeamCmd(t *testing.T, teamID types.TeamID, name, teamURL, avatarURL string) *commandspb.UpdateTeam {
	t.Helper()

	return &commandspb.UpdateTeam{
		TeamId:    string(teamID),
		Name:      ptr.From(name),
		TeamUrl:   ptr.From(teamURL),
		AvatarUrl: ptr.From(avatarURL),
	}
}

func joinTeamCmd(t *testing.T, teamID types.TeamID) *commandspb.JoinTeam {
	t.Helper()

	return &commandspb.JoinTeam{
		TeamId: string(teamID),
	}
}
