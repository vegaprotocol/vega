package teams_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/teams"
	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine(t *testing.T) {
	t.Run("Administrate a team succeeds", testAdministrateTeamSucceeds)
	t.Run("Joining team succeeds", testJoiningTeamSucceeds)
}

func testAdministrateTeamSucceeds(t *testing.T) {
	engine := newEngine(t)

	referrer1 := newPartyID(t)
	name := vgrand.RandomStr(5)
	teamURL := "https://" + name + ".io"
	avatarURL := "https://avatar." + name + ".io"

	teamID1, err := engine.CreateTeam(referrer1, name, teamURL, avatarURL, true)
	require.NoError(t, err)
	assert.NotEmpty(t, teamID1)
	assert.Len(t, teamID1, 64)

	assert.Equal(t, []types.Team{
		{
			ID:            teamID1,
			Referrer:      referrer1,
			Name:          name,
			TeamURL:       teamURL,
			AvatarURL:     avatarURL,
			EnableRewards: true,
		},
	}, engine.ListTeams())

	referrer2 := newPartyID(t)

	// Using same name, team URL and avatar URL as the first team is permitted.
	teamID2, err := engine.CreateTeam(referrer2, name, teamURL, avatarURL, true)
	require.NoError(t, err)
	assert.NotEmpty(t, teamID2)
	assert.Len(t, teamID2, 64)

	assert.NotEqual(t, teamID1, teamID2, "Creating a team should generate an unique ID")

	assertEqualTeams(t, []types.Team{
		{
			ID:            teamID1,
			Referrer:      referrer1,
			Name:          name,
			TeamURL:       teamURL,
			AvatarURL:     avatarURL,
			EnableRewards: true,
		}, {
			ID:            teamID2,
			Referrer:      referrer2,
			Name:          name,
			TeamURL:       teamURL,
			AvatarURL:     avatarURL,
			EnableRewards: true,
		},
	}, engine.ListTeams())

	// A party can only create one team.
	teamID3, err := engine.CreateTeam(referrer2, name, teamURL, avatarURL, true)
	require.EqualError(t, err, teams.ErrPartyAlreadyBelongsToTeam(referrer2).Error())
	assert.Empty(t, teamID3)

	assertEqualTeams(t, []types.Team{
		{
			ID:            teamID1,
			Referrer:      referrer1,
			Name:          name,
			TeamURL:       teamURL,
			AvatarURL:     avatarURL,
			EnableRewards: true,
		}, {
			ID:            teamID2,
			Referrer:      referrer2,
			Name:          name,
			TeamURL:       teamURL,
			AvatarURL:     avatarURL,
			EnableRewards: true,
		},
	}, engine.ListTeams())

	referrer3 := newPartyID(t)

	// Name, team URL and avatar URL are optional.
	teamID4, err := engine.CreateTeam(referrer3, "", "", "", true)
	require.NoError(t, err)
	assert.NotEmpty(t, teamID4)
	assert.Len(t, teamID4, 64)

	assert.NotEqual(t, teamID1, teamID4, "Creating a team should generate an unique ID")
	assert.NotEqual(t, teamID2, teamID4, "Creating a team should generate an unique ID")

	assertEqualTeams(t, []types.Team{
		{
			ID:            teamID1,
			Referrer:      referrer1,
			Name:          name,
			TeamURL:       teamURL,
			AvatarURL:     avatarURL,
			EnableRewards: true,
		}, {
			ID:            teamID2,
			Referrer:      referrer2,
			Name:          name,
			TeamURL:       teamURL,
			AvatarURL:     avatarURL,
			EnableRewards: true,
		}, {
			ID:            teamID4,
			Referrer:      referrer3,
			Name:          "",
			TeamURL:       "",
			AvatarURL:     "",
			EnableRewards: true,
		},
	}, engine.ListTeams())

	// Updating first team
	updatedName := vgrand.RandomStr(5)
	updatedTeamURL := "https://" + name + ".io"
	updatedAvatarURL := "https://avatar." + name + ".io"

	require.NoError(t, engine.UpdateTeam(teamID1, updatedName, updatedTeamURL, updatedAvatarURL, false))

	assertEqualTeams(t, []types.Team{
		{
			ID:            teamID1,
			Referrer:      referrer1,
			Name:          updatedName,
			TeamURL:       updatedTeamURL,
			AvatarURL:     updatedAvatarURL,
			EnableRewards: false,
		}, {
			ID:            teamID2,
			Referrer:      referrer2,
			Name:          name,
			TeamURL:       teamURL,
			AvatarURL:     avatarURL,
			EnableRewards: true,
		}, {
			ID:            teamID4,
			Referrer:      referrer3,
			Name:          "",
			TeamURL:       "",
			AvatarURL:     "",
			EnableRewards: true,
		},
	}, engine.ListTeams())

	unknownTeamID := types.NewTeamID()
	require.EqualError(t,
		engine.UpdateTeam(unknownTeamID, updatedName, updatedTeamURL, updatedAvatarURL, false),
		teams.ErrNoTeamMatchesID(unknownTeamID).Error(),
	)
}

func testJoiningTeamSucceeds(t *testing.T) {
	engine := newEngine(t)

	teamID1, referrer1 := newTeam(t, engine)
	teamID2, referrer2 := newTeam(t, engine)

	require.ErrorIs(t, engine.JoinTeam(teamID1, referrer1), teams.ErrReferrerCannotJoinAnotherTeam)
	require.ErrorIs(t, engine.JoinTeam(teamID1, referrer2), teams.ErrReferrerCannotJoinAnotherTeam)

	referee1 := newPartyID(t)
	require.NoError(t, engine.JoinTeam(teamID1, referee1))
	require.True(t, engine.IsTeamMember(referee1))

	referee2 := newPartyID(t)

	// referee2 tries to join a non-existing team.
	unknownTeamID := types.NewTeamID()
	require.EqualError(t,
		engine.JoinTeam(unknownTeamID, referee2),
		teams.ErrNoTeamMatchesID(unknownTeamID).Error(),
	)
	require.False(t, engine.IsTeamMember(referee2))

	// referee2 joins an existing team.
	require.NoError(t, engine.JoinTeam(teamID1, referee2))
	require.True(t, engine.IsTeamMember(referee2))

	// referee2 just joined another team and want to move on next epoch.
	require.NoError(t, engine.JoinTeam(teamID2, referee2))
	require.True(t, engine.IsTeamMember(referee2))

	// This shows the referee2 joined the first team he applied to, despite
	// his second application to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID:            teamID1,
			Referrer:      referrer1,
			Referees:      []types.PartyID{referee1, referee2},
			EnableRewards: true,
		}, {
			ID:            teamID2,
			Referrer:      referrer2,
			EnableRewards: true,
		},
	}, engine.ListTeams())

	// Simulating end of epoch.
	endEpoch(t, engine)

	// This shows the referee2 moved from team 1 to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID:            teamID1,
			Referrer:      referrer1,
			Referees:      []types.PartyID{referee1},
			EnableRewards: true,
		}, {
			ID:            teamID2,
			Referrer:      referrer2,
			Referees:      []types.PartyID{referee2},
			EnableRewards: true,
		},
	}, engine.ListTeams())
}
