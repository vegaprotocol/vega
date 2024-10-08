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

	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/paths"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTakingAndRestoringSnapshotSucceeds(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)

	vegaPath := paths.New(t.TempDir())
	now := time.Now()

	te1 := newEngine(t)
	snapshotEngine1 := newSnapshotEngine(t, vegaPath, now, te1.engine)
	closeSnapshotEngine1 := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer closeSnapshotEngine1()

	require.NoError(t, snapshotEngine1.Start(ctx))

	teamID1 := newTeamID(t)
	referrer1 := newPartyID(t)
	name1 := vgrand.RandomStr(5)
	teamURL1 := "https://" + name1 + ".io"
	avatarURL1 := "https://avatar." + name1 + ".io"

	expectTeamCreatedEvent(t, te1)
	team1CreationDate := time.Now()
	te1.timeService.EXPECT().GetTimeNow().Return(team1CreationDate).Times(1)

	require.NoError(t, te1.engine.CreateTeam(ctx, referrer1, teamID1, createTeamCmd(t, name1, teamURL1, avatarURL1)))

	teamID2 := newTeamID(t)
	referrer2 := newPartyID(t)
	name2 := vgrand.RandomStr(5)
	teamURL2 := "https://" + name2 + ".io"
	avatarURL2 := "https://avatar." + name2 + ".io"

	expectTeamCreatedEvent(t, te1)
	team2CreationDate := time.Now()
	te1.timeService.EXPECT().GetTimeNow().Return(team2CreationDate).Times(1)
	require.NoError(t, te1.engine.CreateTeam(ctx, referrer2, teamID2, createTeamCmd(t, name2, teamURL2, avatarURL2)))

	referee1 := newPartyID(t)
	expectRefereeJoinedTeamEvent(t, te1)
	referee1JoiningDate := time.Now()
	te1.timeService.EXPECT().GetTimeNow().Return(referee1JoiningDate).Times(1)
	require.NoError(t, te1.engine.JoinTeam(ctx, referee1, joinTeamCmd(t, teamID1)))

	referee2 := newPartyID(t)

	expectRefereeJoinedTeamEvent(t, te1)
	referee2JoiningDate := time.Now()
	te1.timeService.EXPECT().GetTimeNow().Return(referee2JoiningDate).Times(1)
	require.NoError(t, te1.engine.JoinTeam(ctx, referee2, joinTeamCmd(t, teamID1)))

	// This will occur on next epoch, after the snapshot. This help to ensure
	// team switches are properly snapshot.
	require.NoError(t, te1.engine.JoinTeam(ctx, referee2, joinTeamCmd(t, teamID2)))
	require.True(t, te1.engine.IsTeamMember(referee2))

	referee3 := newPartyID(t)

	// Closing the team2 to check the allow list is properly snapshot.
	expectTeamUpdatedEvent(t, te1)
	require.NoError(t, te1.engine.UpdateTeam(ctx, referrer2, teamID2, updateTeamCmd(t, name2, teamURL2, avatarURL2, true, []string{referee3.String()})))

	// Take a snapshot.
	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	expectRefereeSwitchedTeamEvent(t, te1)
	referee2JoiningDate2 := time.Now()
	nextEpoch(t, ctx, te1, referee2JoiningDate2)

	expectRefereeJoinedTeamEvent(t, te1)
	referee3JoiningDate := time.Now()
	te1.timeService.EXPECT().GetTimeNow().Return(referee3JoiningDate).Times(1)
	require.NoError(t, te1.engine.JoinTeam(ctx, referee3, joinTeamCmd(t, teamID2)))

	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te1.currentEpoch - 1,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee1,
					JoinedAt:       referee1JoiningDate,
					StartedAtEpoch: te1.currentEpoch - 1,
				},
			},
			Name:      name1,
			TeamURL:   teamURL1,
			AvatarURL: avatarURL1,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te1.currentEpoch - 1,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee2,
					JoinedAt:       referee2JoiningDate2,
					StartedAtEpoch: te1.currentEpoch,
				},
				{
					PartyID:        referee3,
					JoinedAt:       referee3JoiningDate,
					StartedAtEpoch: te1.currentEpoch,
				},
			},
			Name:      name2,
			TeamURL:   teamURL2,
			AvatarURL: avatarURL2,
			CreatedAt: team2CreationDate,
			Closed:    true,
			AllowList: []types.PartyID{referee3},
		},
	}, te1.engine.ListTeams())

	state1 := map[string][]byte{}
	for _, key := range te1.engine.Keys() {
		state, additionalProvider, err := te1.engine.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state1[key] = state
	}

	closeSnapshotEngine1()

	// Reload the engine using the previous snapshot.

	te2 := newEngine(t)
	snapshotEngine2 := newSnapshotEngine(t, vegaPath, now, te2.engine)
	defer snapshotEngine2.Close()

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, snapshotEngine2.Start(ctx))

	// Comparing the hash after restoration, to ensure it produces the same result.
	hash2, _, _ := snapshotEngine2.Info()
	require.Equal(t, hash1, hash2)

	expectRefereeSwitchedTeamEvent(t, te2)
	nextEpoch(t, ctx, te2, referee2JoiningDate2)

	expectRefereeJoinedTeamEvent(t, te2)
	te2.timeService.EXPECT().GetTimeNow().Return(referee3JoiningDate).Times(1)
	require.NoError(t, te2.engine.JoinTeam(ctx, referee3, joinTeamCmd(t, teamID2)))

	assertEqualTeams(t, []types.Team{
		{
			ID: teamID1,
			Referrer: &types.Membership{
				PartyID:        referrer1,
				JoinedAt:       team1CreationDate,
				StartedAtEpoch: te2.currentEpoch - 1,
			},
			Referees: []*types.Membership{
				{
					PartyID:        referee1,
					JoinedAt:       referee1JoiningDate,
					StartedAtEpoch: te2.currentEpoch - 1,
				},
			},
			Name:      name1,
			TeamURL:   teamURL1,
			AvatarURL: avatarURL1,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:        referrer2,
				JoinedAt:       team2CreationDate,
				StartedAtEpoch: te2.currentEpoch - 1,
			},
			Name:      name2,
			TeamURL:   teamURL2,
			AvatarURL: avatarURL2,
			Referees: []*types.Membership{
				{
					PartyID:        referee2,
					JoinedAt:       referee2JoiningDate2,
					StartedAtEpoch: te2.currentEpoch,
				},
				{
					PartyID:        referee3,
					JoinedAt:       referee3JoiningDate,
					StartedAtEpoch: te2.currentEpoch,
				},
			},
			CreatedAt: team2CreationDate,
			Closed:    true,
			AllowList: []types.PartyID{referee3},
		},
	}, te2.engine.ListTeams())

	state2 := map[string][]byte{}
	for _, key := range te2.engine.Keys() {
		state, additionalProvider, err := te2.engine.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state2[key] = state
	}

	for key := range state1 {
		assert.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}
