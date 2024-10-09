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
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEngine struct {
	engine       *teams.SnapshottedEngine
	broker       *mocks.MockBroker
	timeService  *mocks.MockTimeService
	currentEpoch uint64
}

func assertEqualTeams(t *testing.T, expected, actual []types.Team) {
	t.Helper()

	teams.SortByTeamID(expected)
	teams.SortByTeamID(actual)

	if len(expected) != len(actual) {
		assert.Fail(t, fmt.Sprintf("Expected len of %d but got %d", len(expected), len(actual)))
	}

	for i := 0; i < len(expected); i++ {
		t.Run(fmt.Sprintf("team #%d", i), func(tt *testing.T) {
			expectedTeam := expected[i]
			actualTeam := actual[i]
			assert.Equal(tt, expectedTeam.ID, actualTeam.ID)
			assert.Equal(tt, expectedTeam.Name, actualTeam.Name)
			assert.Equal(tt, expectedTeam.TeamURL, actualTeam.TeamURL)
			assert.Equal(tt, expectedTeam.AvatarURL, actualTeam.AvatarURL)
			assert.Equal(tt, expectedTeam.CreatedAt.UnixNano(), actualTeam.CreatedAt.UnixNano())
			assert.Equal(tt, expectedTeam.Closed, actualTeam.Closed)
			assert.Equal(tt, expectedTeam.AllowList, actualTeam.AllowList)
			assertEqualMembership(tt, expectedTeam.Referrer, actualTeam.Referrer)

			if len(expectedTeam.Referees) != len(actualTeam.Referees) {
				assert.Fail(tt, fmt.Sprintf("number of referees in expected and actual results mismatch, expecting %d but got %d", len(expectedTeam.Referees), len(actualTeam.Referees)))
				return
			}

			for j := 0; j < len(expectedTeam.Referees); j++ {
				tt.Run(fmt.Sprintf("referee #%d", j), func(ttt *testing.T) {
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
	assert.Equal(t, expected.StartedAtEpoch, actual.StartedAtEpoch)
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
		Seq:     te.currentEpoch,
		Action:  vegapb.EpochAction_EPOCH_ACTION_END,
		EndTime: startEpochTime.Add(-1 * time.Second),
	})

	te.currentEpoch += 1
	te.engine.OnEpoch(ctx, types.Epoch{
		Seq:       te.currentEpoch,
		Action:    vegapb.EpochAction_EPOCH_ACTION_START,
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

	broker := mocks.NewMockBroker(ctrl)
	timeService := mocks.NewMockTimeService(ctrl)

	engine := teams.NewSnapshottedEngine(broker, timeService)

	engine.OnEpochRestore(context.Background(), types.Epoch{
		Seq:    10,
		Action: vegapb.EpochAction_EPOCH_ACTION_START,
	})

	return &testEngine{
		engine:       engine,
		broker:       broker,
		timeService:  timeService,
		currentEpoch: 10,
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

func newTeam(t *testing.T, ctx context.Context, te *testEngine) (types.TeamID, types.PartyID, string) {
	t.Helper()

	teamID := newTeamID(t)
	referrer := newPartyID(t)
	teamName := vgrand.RandomStr(5)

	expectTeamCreatedEvent(t, te)

	err := te.engine.CreateTeam(ctx, referrer, teamID, createTeamCmd(t, teamName, "", ""))
	require.NoError(t, err)
	require.NotEmpty(t, teamID)
	require.True(t, te.engine.IsTeamMember(referrer))

	return teamID, referrer, teamName
}

func newTeamWithCmd(t *testing.T, ctx context.Context, te *testEngine, cmd *commandspb.CreateReferralSet_Team) (types.TeamID, types.PartyID) {
	t.Helper()

	teamID := newTeamID(t)
	referrer := newPartyID(t)

	expectTeamCreatedEvent(t, te)

	err := te.engine.CreateTeam(ctx, referrer, teamID, cmd)
	require.NoError(t, err)
	require.NotEmpty(t, teamID)
	require.True(t, te.engine.IsTeamMember(referrer))

	return teamID, referrer
}

func createTeamCmd(t *testing.T, name, teamURL, avatarURL string) *commandspb.CreateReferralSet_Team {
	t.Helper()

	return &commandspb.CreateReferralSet_Team{
		Name:      name,
		TeamUrl:   ptr.From(teamURL),
		AvatarUrl: ptr.From(avatarURL),
	}
}

func createTeamWithAllowListCmd(t *testing.T, name, teamURL, avatarURL string, closed bool, allowList []string) *commandspb.CreateReferralSet_Team {
	t.Helper()

	return &commandspb.CreateReferralSet_Team{
		Name:      name,
		TeamUrl:   ptr.From(teamURL),
		AvatarUrl: ptr.From(avatarURL),
		Closed:    closed,
		AllowList: allowList,
	}
}

func updateTeamCmd(t *testing.T, name, teamURL, avatarURL string, closed bool, allowList []string) *commandspb.UpdateReferralSet_Team {
	t.Helper()

	return &commandspb.UpdateReferralSet_Team{
		Name:      ptr.From(name),
		TeamUrl:   ptr.From(teamURL),
		AvatarUrl: ptr.From(avatarURL),
		Closed:    ptr.From(closed),
		AllowList: allowList,
	}
}

func joinTeamCmd(t *testing.T, teamID types.TeamID) *commandspb.JoinTeam {
	t.Helper()

	return &commandspb.JoinTeam{
		Id: string(teamID),
	}
}
