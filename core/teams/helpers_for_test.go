package teams_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/teams"
	"code.vegaprotocol.io/vega/core/teams/mocks"
	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	typespb "code.vegaprotocol.io/vega/protos/vega"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertEqualTeams(t *testing.T, expected []types.Team, actual []types.Team) {
	t.Helper()

	teams.SortByTeamID(expected)
	assert.Equal(t, expected, actual)
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

func endEpoch(t *testing.T, ctx context.Context, engine *testEngine) {
	t.Helper()

	engine.OnEpoch(ctx, types.Epoch{
		Action: typespb.EpochAction_EPOCH_ACTION_END,
	})
}

func newEngine(t *testing.T) *testEngine {
	t.Helper()

	ctrl := gomock.NewController(t)

	epochEngine := mocks.NewMockEpochEngine(ctrl)
	epochEngine.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any())

	broker := mocks.NewMockBroker(ctrl)

	engine := teams.NewEngine(epochEngine, broker)
	return &testEngine{
		Engine: engine,
		broker: broker,
	}
}

func newPartyID(t *testing.T) types.PartyID {
	t.Helper()

	return types.PartyID(vgrand.RandomStr(5))
}

func newTeam(t *testing.T, ctx context.Context, engine *testEngine) (types.TeamID, types.PartyID) {
	t.Helper()

	expectTeamCreatedEvent(t, engine)

	referrer := types.PartyID(vgrand.RandomStr(5))
	teamID, err := engine.CreateTeam(ctx, referrer, "", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, teamID)
	require.True(t, engine.IsTeamMember(referrer))

	return teamID, referrer
}

type testEngine struct {
	*teams.Engine

	broker *mocks.MockBroker
}
