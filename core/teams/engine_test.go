package teams_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/teams"
	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine(t *testing.T) {
	t.Run("Administrate a team succeeds", testAdministrateTeamSucceeds)
	t.Run("Joining team succeeds", testJoiningTeamSucceeds)
}

func testAdministrateTeamSucceeds(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	engine := newEngine(t)

	referrer1 := newPartyID(t)
	name := vgrand.RandomStr(5)
	teamURL := "https://" + name + ".io"
	avatarURL := "https://avatar." + name + ".io"

	expectTeamCreatedEvent(t, engine)

	teamID1, err := engine.CreateTeam(ctx, referrer1, name, teamURL, avatarURL)
	require.NoError(t, err)
	assert.NotEmpty(t, teamID1)
	assert.Len(t, teamID1, 64)

	assert.Equal(t, []types.Team{
		{
			ID:        teamID1,
			Referrer:  referrer1,
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
		},
	}, engine.ListTeams())

	referrer2 := newPartyID(t)

	expectTeamCreatedEvent(t, engine)

	// Using same name, team URL and avatar URL as the first team is permitted.
	teamID2, err := engine.CreateTeam(ctx, referrer2, name, teamURL, avatarURL)
	require.NoError(t, err)
	assert.NotEmpty(t, teamID2)
	assert.Len(t, teamID2, 64)

	assert.NotEqual(t, teamID1, teamID2, "Creating a team should generate an unique ID")

	assertEqualTeams(t, []types.Team{
		{
			ID:        teamID1,
			Referrer:  referrer1,
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
		}, {
			ID:        teamID2,
			Referrer:  referrer2,
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
		},
	}, engine.ListTeams())

	// A party can only create one team.
	teamID3, err := engine.CreateTeam(ctx, referrer2, name, teamURL, avatarURL)
	require.EqualError(t, err, teams.ErrPartyAlreadyBelongsToTeam(referrer2).Error())
	assert.Empty(t, teamID3)

	assertEqualTeams(t, []types.Team{
		{
			ID:        teamID1,
			Referrer:  referrer1,
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
		}, {
			ID:        teamID2,
			Referrer:  referrer2,
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
		},
	}, engine.ListTeams())

	referrer3 := newPartyID(t)

	expectTeamCreatedEvent(t, engine)

	// Name, team URL and avatar URL are optional.
	teamID4, err := engine.CreateTeam(ctx, referrer3, "", "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, teamID4)
	assert.Len(t, teamID4, 64)

	assert.NotEqual(t, teamID1, teamID4, "Creating a team should generate an unique ID")
	assert.NotEqual(t, teamID2, teamID4, "Creating a team should generate an unique ID")

	assertEqualTeams(t, []types.Team{
		{
			ID:        teamID1,
			Referrer:  referrer1,
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
		}, {
			ID:        teamID2,
			Referrer:  referrer2,
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
		}, {
			ID:        teamID4,
			Referrer:  referrer3,
			Name:      "",
			TeamURL:   "",
			AvatarURL: "",
		},
	}, engine.ListTeams())

	// Updating first team
	updatedName := vgrand.RandomStr(5)
	updatedTeamURL := "https://" + name + ".io"
	updatedAvatarURL := "https://avatar." + name + ".io"

	expectTeamUpdatedEvent(t, engine)

	require.NoError(t, engine.UpdateTeam(ctx, teamID1, updatedName, updatedTeamURL, updatedAvatarURL))

	assertEqualTeams(t, []types.Team{
		{
			ID:        teamID1,
			Referrer:  referrer1,
			Name:      updatedName,
			TeamURL:   updatedTeamURL,
			AvatarURL: updatedAvatarURL,
		}, {
			ID:        teamID2,
			Referrer:  referrer2,
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
		}, {
			ID:        teamID4,
			Referrer:  referrer3,
			Name:      "",
			TeamURL:   "",
			AvatarURL: "",
		},
	}, engine.ListTeams())

	unknownTeamID := types.NewTeamID()
	require.EqualError(t,
		engine.UpdateTeam(ctx, unknownTeamID, updatedName, updatedTeamURL, updatedAvatarURL),
		teams.ErrNoTeamMatchesID(unknownTeamID).Error(),
	)
}

func testJoiningTeamSucceeds(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	engine := newEngine(t)

	teamID1, referrer1 := newTeam(t, ctx, engine)
	teamID2, referrer2 := newTeam(t, ctx, engine)

	require.ErrorIs(t, engine.JoinTeam(ctx, teamID1, referrer1), teams.ErrReferrerCannotJoinAnotherTeam)
	require.ErrorIs(t, engine.JoinTeam(ctx, teamID1, referrer2), teams.ErrReferrerCannotJoinAnotherTeam)

	referee1 := newPartyID(t)
	expectRefereeJoinedTeamEvent(t, engine)
	require.NoError(t, engine.JoinTeam(ctx, teamID1, referee1))
	require.True(t, engine.IsTeamMember(referee1))

	referee2 := newPartyID(t)

	// referee2 tries to join a non-existing team.
	unknownTeamID := types.NewTeamID()
	require.EqualError(t,
		engine.JoinTeam(ctx, unknownTeamID, referee2),
		teams.ErrNoTeamMatchesID(unknownTeamID).Error(),
	)
	require.False(t, engine.IsTeamMember(referee2))

	// referee2 joins an existing team.
	expectRefereeJoinedTeamEvent(t, engine)
	require.NoError(t, engine.JoinTeam(ctx, teamID1, referee2))
	require.True(t, engine.IsTeamMember(referee2))

	// referee2 just joined another team and want to move on next epoch.
	require.NoError(t, engine.JoinTeam(ctx, teamID2, referee2))
	require.True(t, engine.IsTeamMember(referee2))

	// This shows the referee2 joined the first team he applied to, despite
	// his second application to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID:       teamID1,
			Referrer: referrer1,
			Referees: []types.PartyID{referee1, referee2},
		}, {
			ID:       teamID2,
			Referrer: referrer2,
		},
	}, engine.ListTeams())

	// Simulating end of epoch.
	expectRefereeSwitchedTeamEvent(t, engine)
	endEpoch(t, ctx, engine)

	// This shows the referee2 moved from team 1 to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID:       teamID1,
			Referrer: referrer1,
			Referees: []types.PartyID{referee1},
		}, {
			ID:       teamID2,
			Referrer: referrer2,
			Referees: []types.PartyID{referee2},
		},
	}, engine.ListTeams())

	// referee2 just re-joins team 1.
	require.NoError(t, engine.JoinTeam(ctx, teamID1, referee2))
	require.True(t, engine.IsTeamMember(referee2))

	// Simulating end of epoch.
	expectRefereeSwitchedTeamEvent(t, engine)
	endEpoch(t, ctx, engine)

	// This shows the referee2 moved from team 1 to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID:       teamID1,
			Referrer: referrer1,
			Referees: []types.PartyID{referee1, referee2},
		}, {
			ID:       teamID2,
			Referrer: referrer2,
			Referees: []types.PartyID{},
		},
	}, engine.ListTeams())
}
