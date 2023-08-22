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

	// Take a snapshot.
	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	expectRefereeSwitchedTeamEvent(t, te1)
	referee2JoiningDate2 := time.Now()
	nextEpoch(t, ctx, te1, referee2JoiningDate2)

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
			Name:      name1,
			TeamURL:   teamURL1,
			AvatarURL: avatarURL1,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:       referrer2,
				JoinedAt:      team2CreationDate,
				NumberOfEpoch: 1,
			},
			Name:      name2,
			TeamURL:   teamURL2,
			AvatarURL: avatarURL2,
			Referees: []*types.Membership{
				{
					PartyID:       referee2,
					JoinedAt:      referee2JoiningDate2,
					NumberOfEpoch: 0,
				},
			},
			CreatedAt: team2CreationDate,
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
			Name:      name1,
			TeamURL:   teamURL1,
			AvatarURL: avatarURL1,
			CreatedAt: team1CreationDate,
		}, {
			ID: teamID2,
			Referrer: &types.Membership{
				PartyID:       referrer2,
				JoinedAt:      team2CreationDate,
				NumberOfEpoch: 1,
			},
			Name:      name2,
			TeamURL:   teamURL2,
			AvatarURL: avatarURL2,
			Referees: []*types.Membership{
				{
					PartyID:       referee2,
					JoinedAt:      referee2JoiningDate2,
					NumberOfEpoch: 0,
				},
			},
			CreatedAt: team2CreationDate,
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
