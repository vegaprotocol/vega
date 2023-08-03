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
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	typespb "code.vegaprotocol.io/vega/protos/vega"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertEqualTeams(t *testing.T, expected, actual []types.Team) {
	t.Helper()

	teams.SortByTeamID(expected)

	if len(expected) != len(actual) {
		assert.Fail(t, fmt.Sprintf("Expected len of %d but got %d", len(expected), len(actual)))
	}

	for i := 0; i < len(expected); i++ {
		expectedTeam := expected[i]
		actualTeam := actual[i]
		assert.Equal(t, expectedTeam.ID, actualTeam.ID)
		assert.Equal(t, expectedTeam.Name, actualTeam.Name)
		assert.Equal(t, expectedTeam.TeamURL, actualTeam.TeamURL)
		assert.Equal(t, expectedTeam.AvatarURL, actualTeam.AvatarURL)
		assert.Equal(t, expectedTeam.CreatedAt.UnixNano(), actualTeam.CreatedAt.UnixNano())
		assertEqualMembership(t, expectedTeam.Referrer, actualTeam.Referrer)

		for i := 0; i < len(expectedTeam.Referees); i++ {
			assertEqualMembership(t, expectedTeam.Referees[i], actualTeam.Referees[i])
		}
	}
}

func assertEqualMembership(t *testing.T, expected, actual *types.Membership) {
	t.Helper()

	assert.Equal(t, expected.PartyID, actual.PartyID)
	assert.Equal(t, expected.JoinedAt.UnixNano(), actual.JoinedAt.UnixNano())
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

func endEpoch(t *testing.T, ctx context.Context, te *testEngine) {
	t.Helper()

	te.engine.OnEpoch(ctx, types.Epoch{
		Action: typespb.EpochAction_EPOCH_ACTION_END,
	})
}

func newSnapshotEngine(t *testing.T, vegaPath paths.Paths, now time.Time, engine *teams.SnapshotEngine) *snapshot.Engine {
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

	engine := teams.NewSnapshotEngine(epochEngine, broker, timeService)

	return &testEngine{
		engine:      engine,
		broker:      broker,
		timeService: timeService,
	}
}

func newPartyID(t *testing.T) types.PartyID {
	t.Helper()

	return types.PartyID(vgrand.RandomStr(5))
}

func newTeam(t *testing.T, ctx context.Context, te *testEngine) (types.TeamID, types.PartyID) {
	t.Helper()

	expectTeamCreatedEvent(t, te)

	referrer := types.PartyID(vgrand.RandomStr(5))
	teamID, err := te.engine.CreateTeam(ctx, referrer, "", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, teamID)
	require.True(t, te.engine.IsTeamMember(referrer))

	return teamID, referrer
}

type testEngine struct {
	engine      *teams.SnapshotEngine
	broker      *mocks.MockBroker
	timeService *mocks.MockTimeService
}
