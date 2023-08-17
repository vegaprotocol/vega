package teams_test

import (
	"testing"
	"time"

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

	te := newEngine(t)

	referrer1 := newPartyID(t)
	teamID1 := newTeamID(t)
	name := vgrand.RandomStr(5)
	teamURL := "https://" + name + ".io"
	avatarURL := "https://avatar." + name + ".io"

	expectTeamCreatedEvent(t, te)

	team1CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team1CreationDate).Times(1)

	require.NoError(t, te.engine.CreateTeam(ctx, referrer1, teamID1, createTeamCmd(t, name, teamURL, avatarURL)))

	assert.Equal(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:  referrer1,
				JoinedAt: team1CreationDate,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team1CreationDate,
		},
	}, te.engine.ListTeams())

	referrer2 := newPartyID(t)
	teamID2 := newTeamID(t)

	expectTeamCreatedEvent(t, te)

	team2CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team2CreationDate).Times(1)

	// Using same name, team URL and avatar URL as the first team is permitted.
	require.NoError(t, te.engine.CreateTeam(ctx, referrer2, teamID2, createTeamCmd(t, name, teamURL, avatarURL)))

	assert.NotEqual(t, teamID1, teamID2, "Creating a team should generate an unique ID")

	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:  referrer1,
				JoinedAt: team1CreationDate,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:  referrer2,
				JoinedAt: team2CreationDate,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team2CreationDate,
		},
	}, te.engine.ListTeams())

	teamID3 := newTeamID(t)

	// A party can only create one team.
	require.EqualError(t,
		te.engine.CreateTeam(ctx, referrer2, teamID3, createTeamCmd(t, name, teamURL, avatarURL)),
		teams.ErrPartyAlreadyBelongsToTeam(referrer2).Error(),
	)

	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:  referrer1,
				JoinedAt: team1CreationDate,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:  referrer2,
				JoinedAt: team2CreationDate,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team2CreationDate,
		},
	}, te.engine.ListTeams())

	referrer3 := newPartyID(t)
	teamID4 := newTeamID(t)

	expectTeamCreatedEvent(t, te)

	team4CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team4CreationDate).Times(1)

	// Name, team URL and avatar URL are optional.
	require.NoError(t, te.engine.CreateTeam(ctx, referrer3, teamID4, createTeamCmd(t, "", "", "")))

	assert.NotEqual(t, teamID1, teamID4, "Creating a team should generate an unique ID")
	assert.NotEqual(t, teamID2, teamID4, "Creating a team should generate an unique ID")

	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:  referrer1,
				JoinedAt: team1CreationDate,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:  referrer2,
				JoinedAt: team2CreationDate,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team2CreationDate,
		}, {
			ID: teamID4,
			Referrer: &types.Membership{
				PartyID:  referrer3,
				JoinedAt: team4CreationDate,
			},
			Referees:  nil,
			Name:      "",
			TeamURL:   "",
			AvatarURL: "",
			CreatedAt: team4CreationDate,
		},
	}, te.engine.ListTeams())

	// Updating first team
	updatedName := vgrand.RandomStr(5)
	updatedTeamURL := "https://" + name + ".io"
	updatedAvatarURL := "https://avatar." + name + ".io"

	expectTeamUpdatedEvent(t, te)

	require.NoError(t, te.engine.UpdateTeam(ctx, referrer1, updateTeamCmd(t, teamID1, updatedName, updatedTeamURL, updatedAvatarURL)))

	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:  referrer1,
				JoinedAt: team1CreationDate,
			},
			Name:      updatedName,
			TeamURL:   updatedTeamURL,
			AvatarURL: updatedAvatarURL,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:  referrer2,
				JoinedAt: team2CreationDate,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team2CreationDate,
		}, {
			ID: teamID4,
			Referrer: &types.Membership{
				PartyID:  referrer3,
				JoinedAt: team4CreationDate,
			},
			Name:      "",
			TeamURL:   "",
			AvatarURL: "",
			CreatedAt: team4CreationDate,
		},
	}, te.engine.ListTeams())

	unknownTeamID := types.NewTeamID()
	require.EqualError(t,
		te.engine.UpdateTeam(ctx, referrer1, updateTeamCmd(t, unknownTeamID, updatedName, updatedTeamURL, updatedAvatarURL)),
		teams.ErrNoTeamMatchesID(unknownTeamID).Error(),
	)

	require.ErrorIs(t,
		teams.ErrOnlyReferrerCanUpdateTeam,
		te.engine.UpdateTeam(ctx, referrer2, updateTeamCmd(t, teamID1, updatedName, updatedTeamURL, updatedAvatarURL)),
	)
}

func testJoiningTeamSucceeds(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)

	team1CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team1CreationDate).Times(1)
	teamID1, referrer1 := newTeam(t, ctx, te)

	team2CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team2CreationDate).Times(1)
	teamID2, referrer2 := newTeam(t, ctx, te)

	require.ErrorIs(t, te.engine.JoinTeam(ctx, referrer1, joinTeamCmd(t, teamID1)), teams.ErrReferrerCannotJoinAnotherTeam)
	require.ErrorIs(t, te.engine.JoinTeam(ctx, referrer2, joinTeamCmd(t, teamID1)), teams.ErrReferrerCannotJoinAnotherTeam)

	referee1 := newPartyID(t)
	referee1JoiningDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(referee1JoiningDate).Times(1)
	expectRefereeJoinedTeamEvent(t, te)
	require.NoError(t, te.engine.JoinTeam(ctx, referee1, joinTeamCmd(t, teamID1)))
	require.True(t, te.engine.IsTeamMember(referee1))

	referee2 := newPartyID(t)

	// referee2 tries to join a non-existing team.
	unknownTeamID := types.NewTeamID()
	require.EqualError(t,
		te.engine.JoinTeam(ctx, referee2, joinTeamCmd(t, unknownTeamID)),
		teams.ErrNoTeamMatchesID(unknownTeamID).Error(),
	)
	require.False(t, te.engine.IsTeamMember(referee2))

	// referee2 joins an existing team.
	expectRefereeJoinedTeamEvent(t, te)
	referee2JoiningDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(referee2JoiningDate).Times(1)
	require.NoError(t, te.engine.JoinTeam(ctx, referee2, joinTeamCmd(t, teamID1)))
	require.True(t, te.engine.IsTeamMember(referee2))

	// referee2 just joined another team and want to move on next epoch.
	require.NoError(t, te.engine.JoinTeam(ctx, referee2, joinTeamCmd(t, teamID2)))
	require.True(t, te.engine.IsTeamMember(referee2))

	// This shows the referee2 joined the first team he applied to, despite
	// his second application to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:  referrer1,
				JoinedAt: team1CreationDate,
			},
			Referees: []*types.Membership{
				{
					PartyID:  referee1,
					JoinedAt: referee1JoiningDate,
				}, {
					PartyID:  referee2,
					JoinedAt: referee2JoiningDate,
				},
			},
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:  referrer2,
				JoinedAt: team2CreationDate,
			},
			CreatedAt: team2CreationDate,
		},
	}, te.engine.ListTeams())

	// Simulating end of epoch.
	expectRefereeSwitchedTeamEvent(t, te)
	referee2JoiningDate2 := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(referee2JoiningDate2).Times(1)
	endEpoch(t, ctx, te)

	// This shows the referee2 moved from team 1 to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:  referrer1,
				JoinedAt: team1CreationDate,
			},
			Referees: []*types.Membership{
				{
					PartyID:  referee1,
					JoinedAt: referee1JoiningDate,
				},
			},
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:  referrer2,
				JoinedAt: team2CreationDate,
			},
			Referees: []*types.Membership{
				{
					PartyID:  referee2,
					JoinedAt: referee2JoiningDate2,
				},
			},
			CreatedAt: team2CreationDate,
		},
	}, te.engine.ListTeams())

	// referee2 just re-joins team 1.
	referee2JoiningDate3 := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(referee2JoiningDate3).Times(1)
	require.NoError(t, te.engine.JoinTeam(ctx, referee2, joinTeamCmd(t, teamID1)))
	require.True(t, te.engine.IsTeamMember(referee2))

	// Simulating end of epoch.
	expectRefereeSwitchedTeamEvent(t, te)
	endEpoch(t, ctx, te)

	// This shows the referee2 moved from team 1 to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:  referrer1,
				JoinedAt: team1CreationDate,
			},
			Referees: []*types.Membership{
				{
					PartyID:  referee1,
					JoinedAt: referee1JoiningDate,
				}, {
					PartyID:  referee2,
					JoinedAt: referee2JoiningDate3,
				},
			},
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:  referrer2,
				JoinedAt: team2CreationDate,
			},
			Referees:  []*types.Membership{},
			CreatedAt: team2CreationDate,
		},
	}, te.engine.ListTeams())
}
