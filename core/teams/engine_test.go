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
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/teams"
	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine(t *testing.T) {
	t.Run("Administrate a team succeeds", testAdministrateTeamSucceeds)
	t.Run("Joining team succeeds", testJoiningTeamSucceeds)
	t.Run("unique team names", testUniqueTeamNames)
	t.Run("must be in a team for the minimum number of epochs", testMinEpochsRequired)
}

func testUniqueTeamNames(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)

	require.False(t, te.engine.TeamExists(newTeamID(t)))

	referrer1 := newPartyID(t)
	referrer2 := newPartyID(t)
	teamID1 := newTeamID(t)
	teamID2 := newTeamID(t)
	name := vgrand.RandomStr(5)
	teamURL := "https://" + name + ".io"
	avatarURL := "https://avatar." + name + ".io"

	expectTeamCreatedEvent(t, te)

	team1CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team1CreationDate).Times(1)

	require.NoError(t, te.engine.CreateTeam(ctx, referrer1, teamID1, createTeamCmd(t, name, teamURL, avatarURL)))
	require.True(t, te.engine.TeamExists(teamID1))

	require.EqualError(t, te.engine.CreateTeam(ctx, referrer2, teamID2, createTeamCmd(t, name, teamURL, avatarURL)),
		teams.ErrTeamNameIsAlreadyInUse.Error(),
	)
}

func testAdministrateTeamSucceeds(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)

	require.False(t, te.engine.TeamExists(newTeamID(t)))

	referrer1 := newPartyID(t)
	teamID1 := newTeamID(t)
	name := vgrand.RandomStr(5)
	name2 := vgrand.RandomStr(5)
	name3 := vgrand.RandomStr(5)
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
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te.currentEpoch,
			},
			Referees:  nil,
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team1CreationDate,
			Closed:    false,
			AllowList: nil,
		},
	}, te.engine.ListTeams())

	referrer2 := newPartyID(t)
	teamID2 := newTeamID(t)

	expectTeamCreatedEvent(t, te)

	team2CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team2CreationDate).Times(1)

	// Using same team URL and avatar URL as the first team is permitted.
	require.NoError(t, te.engine.CreateTeam(ctx, referrer2, teamID2, createTeamCmd(t, name2, teamURL, avatarURL)))
	require.True(t, te.engine.TeamExists(teamID2))
	assert.NotEqual(t, teamID1, teamID2, "Creating a team should generate an unique ID")

	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te.currentEpoch,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te.currentEpoch,
			},
			Name:      name2,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team2CreationDate,
		},
	}, te.engine.ListTeams())

	teamID3 := newTeamID(t)

	// A party can only create one team.
	require.EqualError(t,
		te.engine.CreateTeam(ctx, referrer2, teamID3, createTeamCmd(t, name3, teamURL, avatarURL)),
		teams.ErrPartyAlreadyBelongsToTeam(referrer2).Error(),
	)

	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te.currentEpoch,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te.currentEpoch,
			},
			Name:      name2,
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
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te.currentEpoch,
			},
			Name:      name,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te.currentEpoch,
			},
			Name:      name2,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team2CreationDate,
		}, {
			ID: teamID4,
			Referrer: &types.Membership{
				PartyID:        referrer3,
				JoinedAt:       team4CreationDate,
				StartedAtEpoch: te.currentEpoch,
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

	require.NoError(t, te.engine.UpdateTeam(ctx, referrer1, teamID1, updateTeamCmd(t, updatedName, updatedTeamURL, updatedAvatarURL, false, nil)))

	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te.currentEpoch,
			},
			Name:      updatedName,
			TeamURL:   updatedTeamURL,
			AvatarURL: updatedAvatarURL,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te.currentEpoch,
			},
			Name:      name2,
			TeamURL:   teamURL,
			AvatarURL: avatarURL,
			CreatedAt: team2CreationDate,
		}, {
			ID: teamID4,
			Referrer: &types.Membership{
				PartyID:        referrer3,
				JoinedAt:       team4CreationDate,
				StartedAtEpoch: te.currentEpoch,
			},
			Name:      team4Name,
			TeamURL:   "",
			AvatarURL: "",
			CreatedAt: team4CreationDate,
		},
	}, te.engine.ListTeams())

	unknownTeamID := types.NewTeamID()
	require.EqualError(t,
		te.engine.UpdateTeam(ctx, referrer1, unknownTeamID, updateTeamCmd(t, updatedName, updatedTeamURL, updatedAvatarURL, false, nil)),
		teams.ErrNoTeamMatchesID(unknownTeamID).Error(),
	)

	require.ErrorIs(t,
		teams.ErrOnlyReferrerCanUpdateTeam,
		te.engine.UpdateTeam(ctx, referrer2, teamID1, updateTeamCmd(t, updatedName, updatedTeamURL, updatedAvatarURL, false, nil)),
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
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te.currentEpoch,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee1,
					JoinedAt:       referee1JoiningDate,
					StartedAtEpoch: te.currentEpoch,
				}, {
					PartyID:        referee2,
					JoinedAt:       referee2JoiningDate,
					StartedAtEpoch: te.currentEpoch,
				},
			},
			Name:      team1Name,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te.currentEpoch,
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
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te.currentEpoch - 1,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee1,
					JoinedAt:       referee1JoiningDate,
					StartedAtEpoch: te.currentEpoch - 1,
				},
			},
			Name:      team1Name,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te.currentEpoch - 1,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee2,
					JoinedAt:       referee2JoiningDate2,
					StartedAtEpoch: te.currentEpoch,
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
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te.currentEpoch - 2,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee1,
					JoinedAt:       referee1JoiningDate,
					StartedAtEpoch: te.currentEpoch - 2,
				}, {
					PartyID:        referee2,
					JoinedAt:       referee2JoiningDate3,
					StartedAtEpoch: te.currentEpoch,
				},
			},
			Name:      team1Name,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te.currentEpoch - 2,
			},
			Referees:  []*types.Membership{},
			Name:      team2Name,
			CreatedAt: team2CreationDate,
		},
	}, te.engine.ListTeams())

	expectTeamUpdatedEvent(t, te)

	// Closing the team.
	require.NoError(t, te.engine.UpdateTeam(ctx, referrer2, teamID2, updateTeamCmd(t, "", "", "", true, nil)))

	// referee2 try to re-join team 2, but joining a closed team without allow-list is disallow.
	require.Error(t, te.engine.JoinTeam(ctx, referee2, joinTeamCmd(t, teamID2)))

	// Simulating moving to next epoch.
	nextEpoch(t, ctx, te, referee2JoiningDate3)

	// This shows the referee2 stayed in team 1.
	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te.currentEpoch - 3,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee1,
					JoinedAt:       referee1JoiningDate,
					StartedAtEpoch: te.currentEpoch - 3,
				}, {
					PartyID:        referee2,
					JoinedAt:       referee2JoiningDate3,
					StartedAtEpoch: te.currentEpoch - 1,
				},
			},
			Name:      team1Name,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te.currentEpoch - 3,
			},
			Referees:  []*types.Membership{},
			Name:      team2Name,
			CreatedAt: team2CreationDate,
			Closed:    true,
		},
	}, te.engine.ListTeams())

	// Allow referee2 to join the closed team.
	expectTeamUpdatedEvent(t, te)
	require.NoError(t, te.engine.UpdateTeam(ctx, referrer2, teamID2, updateTeamCmd(t, "", "", "", true, []string{referee2.String()})))

	// referee2 can re-join team 2, because that party is specified in allow-list.
	require.NoError(t, te.engine.JoinTeam(ctx, referee2, joinTeamCmd(t, teamID2)))

	// Simulating moving to next epoch.
	expectRefereeSwitchedTeamEvent(t, te)
	nextEpoch(t, ctx, te, referee2JoiningDate3)

	// This shows the referee2 moved to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te.currentEpoch - 4,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee1,
					JoinedAt:       referee1JoiningDate,
					StartedAtEpoch: te.currentEpoch - 4,
				},
			},
			Name:      team1Name,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te.currentEpoch - 4,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee2,
					JoinedAt:       referee2JoiningDate3,
					StartedAtEpoch: te.currentEpoch,
				},
			},
			Name:      team2Name,
			CreatedAt: team2CreationDate,
			Closed:    true,
			AllowList: []types.PartyID{referee2},
		},
	}, te.engine.ListTeams())

	// referee1 cannot join team 2, because that party is not specified in allow-list.
	require.Error(t, te.engine.JoinTeam(ctx, referee1, joinTeamCmd(t, teamID2)))

	// Simulating moving to next epoch.
	nextEpoch(t, ctx, te, referee2JoiningDate3)

	// This shows the referee1 did not moved to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te.currentEpoch - 5,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee1,
					JoinedAt:       referee1JoiningDate,
					StartedAtEpoch: te.currentEpoch - 5,
				},
			},
			Name:      team1Name,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te.currentEpoch - 5,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee2,
					JoinedAt:       referee2JoiningDate3,
					StartedAtEpoch: te.currentEpoch - 1,
				},
			},
			Name:      team2Name,
			CreatedAt: team2CreationDate,
			Closed:    true,
			AllowList: []types.PartyID{referee2},
		},
	}, te.engine.ListTeams())

	referee4 := newPartyID(t)

	team3CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team3CreationDate).Times(1)
	team3Name := vgrand.RandomStr(5)
	teamID3, referrer3 := newTeamWithCmd(t, ctx, te, &commandspb.CreateReferralSet_Team{
		Name:      team3Name,
		Closed:    true,
		AllowList: []string{referee4.String()},
	})

	expectRefereeJoinedTeamEvent(t, te)
	referee4JoiningDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(referee4JoiningDate).Times(1)
	require.NoError(t, te.engine.JoinTeam(ctx, referee4, joinTeamCmd(t, teamID3)))
	require.True(t, te.engine.IsTeamMember(referee4))

	// This shows the referee1 did not moved to team 2.
	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te.currentEpoch - 5,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee1,
					JoinedAt:       referee1JoiningDate,
					StartedAtEpoch: te.currentEpoch - 5,
				},
			},
			Name:      team1Name,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te.currentEpoch - 5,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee2,
					JoinedAt:       referee2JoiningDate3,
					StartedAtEpoch: te.currentEpoch - 1,
				},
			},
			Name:      team2Name,
			CreatedAt: team2CreationDate,
			Closed:    true,
			AllowList: []types.PartyID{referee2},
		}, {
			ID: teamID3,
			Referrer: &types.Membership{
				PartyID:        referrer3,
				JoinedAt:       team3CreationDate,
				StartedAtEpoch: te.currentEpoch,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee4,
					JoinedAt:       referee4JoiningDate,
					StartedAtEpoch: te.currentEpoch,
				},
			},
			Name:      team3Name,
			CreatedAt: team3CreationDate,
			Closed:    true,
			AllowList: []types.PartyID{referee4},
		},
	}, te.engine.ListTeams())
}

func testMinEpochsRequired(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)

	team1CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team1CreationDate).Times(1)
	teamID1, _, _ := newTeam(t, ctx, te)

	// if we set min epochs to 0 then we see the referrer
	members := te.engine.GetTeamMembers(string(teamID1), 0)
	assert.Len(t, members, 1)

	// referrer made a team, but does not get returned as a team member until its been the minimum epochs
	members = te.engine.GetTeamMembers(string(teamID1), 5)
	assert.Len(t, members, 0)

	// move to epoch 11 and add a team member
	te.engine.OnEpoch(ctx, types.Epoch{Seq: 11, Action: vegapb.EpochAction_EPOCH_ACTION_START})

	expectRefereeJoinedTeamEvent(t, te)
	referee := newPartyID(t)
	refereeJoiningDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(refereeJoiningDate).Times(1)
	require.NoError(t, te.engine.JoinTeam(ctx, referee, joinTeamCmd(t, teamID1)))

	// referrer joined at epoch 10, team member at 11 lets move to epoch 15
	te.engine.OnEpoch(ctx, types.Epoch{Seq: 15, Action: vegapb.EpochAction_EPOCH_ACTION_START})
	members = te.engine.GetTeamMembers(string(teamID1), 5)
	assert.Len(t, members, 1)

	// now at epoch 16 both should be there
	te.engine.OnEpoch(ctx, types.Epoch{Seq: 16, Action: vegapb.EpochAction_EPOCH_ACTION_START})
	members = te.engine.GetTeamMembers(string(teamID1), 5)
	assert.Len(t, members, 2)
}

func TestRemoveFromAllowListRemoveFromTheTeam(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)

	require.False(t, te.engine.TeamExists(newTeamID(t)))

	referrer1 := newPartyID(t)
	referee1 := newPartyID(t)
	teamID1 := newTeamID(t)
	name := vgrand.RandomStr(5)
	teamURL := "https://" + name + ".io"
	avatarURL := "https://avatar." + name + ".io"

	// create the team
	expectTeamCreatedEvent(t, te)
	team1CreationDate := time.Now()
	te.timeService.EXPECT().GetTimeNow().Return(team1CreationDate).Times(1)

	te.engine.OnEpoch(ctx, types.Epoch{Seq: 1, Action: vegapb.EpochAction_EPOCH_ACTION_START})

	require.NoError(t, te.engine.CreateTeam(ctx, referrer1, teamID1,
		createTeamWithAllowListCmd(t, name, teamURL, avatarURL, true, []string{referee1.String()})))
	require.True(t, te.engine.TeamExists(teamID1))

	// referee join the team
	expectRefereeJoinedTeamEvent(t, te)
	te.timeService.EXPECT().GetTimeNow().Return(team1CreationDate.Add(10 * time.Second)).Times(1)

	require.NoError(t, te.engine.JoinTeam(ctx, referee1, joinTeamCmd(t, teamID1)))
	require.True(t, te.engine.IsTeamMember(referee1))
	// two members in the team
	assert.Len(t, te.engine.GetTeamMembers(string(teamID1), 0), 2)

	te.engine.OnEpoch(ctx, types.Epoch{Seq: 2, Action: vegapb.EpochAction_EPOCH_ACTION_START})

	// referrer update the team to remove all allowlisted parties
	expectTeamUpdatedEvent(t, te)
	// te.timeService.EXPECT().GetTimeNow().Return(team1CreationDate.Add(20 * time.Second)).Times(1)
	require.NoError(t, te.engine.UpdateTeam(ctx, referrer1, teamID1,
		updateTeamCmd(t, name, teamURL, avatarURL, true, []string{})))
	require.True(t, te.engine.TeamExists(teamID1))

	// move to the next epoch
	expectRefereeSwitchedTeamEvent(t, te)
	te.engine.OnEpoch(ctx, types.Epoch{Seq: 2, Action: vegapb.EpochAction_EPOCH_ACTION_START})
	require.False(t, te.engine.IsTeamMember(referee1))
	// only referrer is team member now
	assert.Len(t, te.engine.GetTeamMembers(string(teamID1), 0), 1)
}
