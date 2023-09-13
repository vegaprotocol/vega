package sqlstore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func setupTeamsTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.Teams, *sqlstore.Parties) {
	t.Helper()

	bs := sqlstore.NewBlocks(connectionSource)
	ts := sqlstore.NewTeams(connectionSource)
	ps := sqlstore.NewParties(connectionSource)

	return bs, ts, ps
}

func setupTeams(t *testing.T, ctx context.Context, bs *sqlstore.Blocks, ps *sqlstore.Parties, ts *sqlstore.Teams) ([]entities.Team, []entities.TeamMember) {
	t.Helper()

	teams := make([]entities.Team, 0, 10)
	teamsHistory := []entities.TeamMember{}

	for i := 0; i < 10; i++ {
		block := addTestBlock(t, ctx, bs)
		referrer := addTestParty(t, ctx, ps, block)
		team := entities.Team{
			ID:             entities.TeamID(helpers.GenerateID()),
			Referrer:       referrer.ID,
			Name:           fmt.Sprintf("Test Team %02d", i+1),
			CreatedAt:      block.VegaTime,
			CreatedAtEpoch: 1,
			VegaTime:       block.VegaTime,
		}
		err := ts.AddTeam(ctx, &team)
		require.NoError(t, err)
		teams = append(teams, team)
		teamsHistory = append(teamsHistory, entities.TeamMember{
			TeamID:        team.ID,
			PartyID:       referrer.ID,
			JoinedAtEpoch: 1,
			JoinedAt:      block.VegaTime,
			VegaTime:      block.VegaTime,
		})

		time.Sleep(10 * time.Millisecond)
	}

	for _, team := range teams {
		block := addTestBlock(t, ctx, bs)
		for i := 0; i < 10; i++ {
			referee := addTestParty(t, ctx, ps, block)
			teamReferee := entities.TeamMember{
				TeamID:        team.ID,
				PartyID:       referee.ID,
				JoinedAt:      block.VegaTime,
				JoinedAtEpoch: 2,
				VegaTime:      block.VegaTime,
			}
			err := ts.RefereeJoinedTeam(ctx, &teamReferee)
			require.NoError(t, err)
			teamsHistory = append(teamsHistory, teamReferee)
		}
		time.Sleep(10 * time.Millisecond)
	}

	switchingReferee := teamsHistory[len(teams)].PartyID

	for i, team := range teams {
		if i == 0 {
			continue
		}

		block := addTestBlock(t, ctx, bs)
		switchTeam := entities.RefereeTeamSwitch{
			FromTeamID:      teams[i-1].ID,
			ToTeamID:        team.ID,
			PartyID:         switchingReferee,
			SwitchedAtEpoch: uint64(3 + i),
			SwitchedAt:      block.VegaTime,
			VegaTime:        block.VegaTime,
		}

		require.NoError(t, ts.RefereeSwitchedTeam(ctx, &switchTeam))

		teamsHistory = append(teamsHistory, entities.TeamMember{
			TeamID:        team.ID,
			PartyID:       switchingReferee,
			JoinedAtEpoch: uint64(3 + i),
			JoinedAt:      block.VegaTime,
			VegaTime:      block.VegaTime,
		})
		time.Sleep(10 * time.Millisecond)
	}

	return teams, teamsHistory
}

func historyForReferee(teamsHistory []entities.TeamMember, party entities.PartyID) []entities.TeamMemberHistory {
	var refereeHistory []entities.TeamMemberHistory

	for _, referee := range teamsHistory {
		if referee.PartyID == party {
			refereeHistory = append(refereeHistory, entities.TeamMemberHistory{
				TeamID:        referee.TeamID,
				JoinedAt:      referee.JoinedAt,
				JoinedAtEpoch: referee.JoinedAtEpoch,
			})
		}
	}
	slices.SortStableFunc(refereeHistory, func(a, b entities.TeamMemberHistory) bool {
		return a.JoinedAtEpoch < b.JoinedAtEpoch
	})

	return refereeHistory
}

func currentRefereesForTeam(teamsHistory []entities.TeamMember, teamID entities.TeamID) []entities.TeamMember {
	currentReferees := currentReferees(teamsHistory)

	currentTeamReferees := []entities.TeamMember{}
	for _, referee := range currentReferees {
		if referee.TeamID == teamID {
			currentTeamReferees = append(currentTeamReferees, referee)
		}
	}

	slices.SortStableFunc(currentTeamReferees, func(a, b entities.TeamMember) bool {
		return a.PartyID < b.PartyID
	})

	return currentTeamReferees
}

func currentReferees(teamsHistory []entities.TeamMember) []entities.TeamMember {
	currentReferees := map[entities.PartyID]entities.TeamMember{}

	for _, teamMember := range teamsHistory {
		// teamMember.JoinedAtEpoch != 1 is a ugly hack to exclude the referrer.
		if teamMember.JoinedAtEpoch == 1 {
			continue
		}

		previousMembership, ok := currentReferees[teamMember.PartyID]
		if ok {
			if previousMembership.JoinedAtEpoch < teamMember.JoinedAtEpoch {
				currentReferees[teamMember.PartyID] = teamMember
			}
		} else {
			currentReferees[teamMember.PartyID] = teamMember
		}
	}

	return maps.Values(currentReferees)
}
