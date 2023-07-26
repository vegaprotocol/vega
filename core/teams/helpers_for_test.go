package teams_test

import (
	"context"
	"testing"

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

func endEpoch(t *testing.T, engine *teams.Engine) {
	t.Helper()

	engine.OnEpoch(context.Background(), types.Epoch{
		Action: typespb.EpochAction_EPOCH_ACTION_END,
	})
}

func newEngine(t *testing.T) *teams.Engine {
	t.Helper()

	ctrl := gomock.NewController(t)

	epochEngine := mocks.NewMockEpochEngine(ctrl)
	epochEngine.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any())

	return teams.NewEngine(epochEngine)
}

func newPartyID(t *testing.T) types.PartyID {
	t.Helper()

	return types.PartyID(vgrand.RandomStr(5))
}

func newTeam(t *testing.T, engine *teams.Engine) (types.TeamID, types.PartyID) {
	t.Helper()

	referrer := types.PartyID(vgrand.RandomStr(5))
	teamID, err := engine.CreateTeam(referrer, "", "", "", true)
	require.NoError(t, err)
	require.NotEmpty(t, teamID)
	require.True(t, engine.IsTeamMember(referrer))

	return teamID, referrer
}
