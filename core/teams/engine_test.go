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

	require.False(t, te.engine.TeamExists(newTeamID(t)))

	referrer1 := newPartyID(t)
	teamID1 := newTeamID(t)
	name := vgrand.RandomStr(5)
	teamURL := "https://" + name + ".io"
	avatarURL := "https://avatar." + name + ".io"

	expectTeamCreatedEvent(t, te)

	team1CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team1CreationDate).Times(1)

	require.NoError(t, te.engine.CreateTeam(ctx, referrer1, teamID1, createTeamCmd(t, name, teamURL, avatarURL)))
	require.True(t, te.engine.TeamExists(teamID1))

	assert.Equal(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:       referrer1,
				JoinedAt:      team1CreationDate,
				NumberOfEpoch: 0,
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
	require.True(t, te.engine.TeamExists(teamID2))
	assert.NotEqual(t, teamID1, teamID2, "Creating a team should generate an unique ID")

	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:       referrer1,
				JoinedAt:      team1CreationDate,
				NumberOfEpoch: 0,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:       referrer2,
				JoinedAt:      team2CreationDate,
				NumberOfEpoch: 0,
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
				PartyID:       referrer1,
				JoinedAt:      team1CreationDate,
				NumberOfEpoch: 0,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:       referrer2,
				JoinedAt:      team2CreationDate,
				NumberOfEpoch: 0,
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

	// Team URL and avatar URL are optional.
	team4Name := vgrand.RandomStr(5)
	require.NoError(t, te.engine.CreateTeam(ctx, referrer3, teamID4, createTeamCmd(t, team4Name, "", "")))
	require.True(t, te.engine.TeamExists(teamID4))
	assert.NotEqual(t, teamID1, teamID4, "Creating a team should generate an unique ID")
	assert.NotEqual(t, teamID2, teamID4, "Creating a team should generate an unique ID")

	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:       referrer1,
				JoinedAt:      team1CreationDate,
				NumberOfEpoch: 0,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:       referrer2,
				JoinedAt:      team2CreationDate,
				NumberOfEpoch: 0,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team2CreationDate,
		}, {
			ID: teamID4,
			Referrer: &types.Membership{
				PartyID:       referrer3,
				JoinedAt:      team4CreationDate,
				NumberOfEpoch: 0,
			},
			Referees:  nil,
			Name:      team4Name,
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

	require.NoError(t, te.engine.UpdateTeam(ctx, referrer1, teamID1, updateTeamCmd(t, updatedName, updatedTeamURL, updatedAvatarURL, false)))

	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:       referrer1,
				JoinedAt:      team1CreationDate,
				NumberOfEpoch: 0,
			},
			Name:      updatedName,
			TeamURL:   updatedTeamURL,
			AvatarURL: updatedAvatarURL,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:       referrer2,
				JoinedAt:      team2CreationDate,
				NumberOfEpoch: 0,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team2CreationDate,
		}, {
			ID: teamID4,
			Referrer: &types.Membership{
				PartyID:       referrer3,
				JoinedAt:      team4CreationDate,
				NumberOfEpoch: 0,
			},
			Name:      team4Name,
			TeamURL:   "",
			AvatarURL: "",
			CreatedAt: team4CreationDate,
		},
	}, te.engine.ListTeams())

	unknownTeamID := types.NewTeamID()
	require.EqualError(t,
		te.engine.UpdateTeam(ctx, referrer1, unknownTeamID, updateTeamCmd(t, updatedName, updatedTeamURL, updatedAvatarURL, false)),
		teams.ErrNoTeamMatchesID(unknownTeamID).Error(),
	)

	require.ErrorIs(t,
		teams.ErrOnlyReferrerCanUpdateTeam,
		te.engine.UpdateTeam(ctx, referrer2, teamID1, updateTeamCmd(t, updatedName, updatedTeamURL, updatedAvatarURL, false)),
	)
}

func testJoiningTeamSucceeds(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)

	team1CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team1CreationDate).Times(1)
	teamID1, referrer1, team1Name := newTeam(t, ctx, te)

	team2CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team2CreationDate).Times(1)
	teamID2, referrer2, team2Name := newTeam(t, ctx, te)

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
				PartyID:       referrer1,
				JoinedAt:      team1CreationDate,
				NumberOfEpoch: 0,
			},
			Referees: []*types.Membership{
				{
					PartyID:       referee1,
					JoinedAt:      referee1JoiningDate,
					NumberOfEpoch: 0,
				}, {
					PartyID:       referee2,
					JoinedAt:      referee2JoiningDate,
					NumberOfEpoch: 0,
				},
			},
			Name:      team1Name,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:       referrer2,
				JoinedAt:      team2CreationDate,
				NumberOfEpoch: 0,
			},
			Name:      team2Name,
			CreatedAt: team2CreationDate,
		},
	}, te.engine.ListTeams())

	// Simulating moving to next epoch.
	expectRefereeSwitchedTeamEvent(t, te)
	referee2JoiningDate2 := time.Now()
	nextEpoch(t, ctx, te, referee2JoiningDate2)

	// This shows the referee2 moved from team 1 to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:       referrer1,
				JoinedAt:      team1CreationDate,
				NumberOfEpoch: 1,
			},
			Referees: []*types.Membership{
				{
					PartyID:       referee1,
					JoinedAt:      referee1JoiningDate,
					NumberOfEpoch: 1,
				},
			},
			Name:      team1Name,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:       referrer2,
				JoinedAt:      team2CreationDate,
				NumberOfEpoch: 1,
			},
			Referees: []*types.Membership{
				{
					PartyID:       referee2,
					JoinedAt:      referee2JoiningDate2,
					NumberOfEpoch: 0,
				},
			},
			Name:      team2Name,
			CreatedAt: team2CreationDate,
		},
	}, te.engine.ListTeams())

	// referee2 just re-joins team 1.
	referee2JoiningDate3 := time.Now()
	require.NoError(t, te.engine.JoinTeam(ctx, referee2, joinTeamCmd(t, teamID1)))
	require.True(t, te.engine.IsTeamMember(referee2))

	// Simulating moving to next epoch.
	expectRefereeSwitchedTeamEvent(t, te)
	nextEpoch(t, ctx, te, referee2JoiningDate3)

	// This shows the referee2 moved from team 1 to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:       referrer1,
				JoinedAt:      team1CreationDate,
				NumberOfEpoch: 2,
			},
			Referees: []*types.Membership{
				{
					PartyID:       referee1,
					JoinedAt:      referee1JoiningDate,
					NumberOfEpoch: 2,
				}, {
					PartyID:       referee2,
					JoinedAt:      referee2JoiningDate3,
					NumberOfEpoch: 0,
				},
			},
			Name:      team1Name,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:       referrer2,
				JoinedAt:      team2CreationDate,
				NumberOfEpoch: 2,
			},
			Referees:  []*types.Membership{},
			Name:      team2Name,
			CreatedAt: team2CreationDate,
		},
	}, te.engine.ListTeams())

	expectTeamUpdatedEvent(t, te)

	require.NoError(t, te.engine.UpdateTeam(ctx, referrer2, teamID2, updateTeamCmd(t, "", "", "", true)))

	// referee2 just re-joins team 2.
	require.Error(t, te.engine.JoinTeam(ctx, referee2, joinTeamCmd(t, teamID2)))
	require.True(t, te.engine.IsTeamMember(referee2))

	// This shows the referee2 moved from team 1 to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:       referrer1,
				JoinedAt:      team1CreationDate,
				NumberOfEpoch: 2,
			},
			Referees: []*types.Membership{
				{
					PartyID:       referee1,
					JoinedAt:      referee1JoiningDate,
					NumberOfEpoch: 2,
				}, {
					PartyID:       referee2,
					JoinedAt:      referee2JoiningDate3,
					NumberOfEpoch: 0,
				},
			},
			Name:      team1Name,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:       referrer2,
				JoinedAt:      team2CreationDate,
				NumberOfEpoch: 2,
			},
			Referees:  []*types.Membership{},
			Name:      team2Name,
			CreatedAt: team2CreationDate,
			Closed:    true,
		},
	}, te.engine.ListTeams())
}
