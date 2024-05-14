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

package sqlstore_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type gameScoresTestStore struct {
	gs *sqlstore.GameScores
}

func newGameScoresTestStore(t *testing.T) *gameScoresTestStore {
	t.Helper()
	return &gameScoresTestStore{
		gs: sqlstore.NewGameScores(connectionSource),
	}
}

func TestInsertPartyScores(t *testing.T) {
	ctx := tempTransaction(t)
	store := newGameScoresTestStore(t)
	now := time.Now()
	gps := entities.GamePartyScore{
		GameID:         "FFFF",
		EpochID:        1,
		PartyID:        "EEEE",
		Score:          num.DecimalOne(),
		StakingBalance: num.DecimalTwo(),
		OpenVolume:     num.DecimalZero(),
		TotalFeesPaid:  num.DecimalFromInt64(4),
		IsEligible:     true,
		VegaTime:       now,
	}

	t.Run("can insert successfully", func(t *testing.T) {
		assert.NoError(t, store.gs.AddPartyScore(ctx, gps))
	})

	team := entities.TeamID("AAAA")
	gps.GameID = "BBBB"
	gps.TeamID = &team
	t.Run("can insert successfully with team", func(t *testing.T) {
		assert.NoError(t, store.gs.AddPartyScore(ctx, gps))
	})

	rank := uint64(2)
	gps.PartyID = "BBBB"
	gps.Rank = &rank
	t.Run("can insert successfully with rank", func(t *testing.T) {
		assert.NoError(t, store.gs.AddPartyScore(ctx, gps))
	})
}

func TestInsertTeamScores(t *testing.T) {
	ctx := tempTransaction(t)
	store := newGameScoresTestStore(t)
	now := time.Now()
	gts := entities.GameTeamScore{
		GameID:   "FFFF",
		EpochID:  1,
		TeamID:   "EEEE",
		Score:    num.DecimalOne(),
		VegaTime: now,
	}

	t.Run("can insert successfully", func(t *testing.T) {
		require.NoError(t, store.gs.AddTeamScore(ctx, gts))
	})
}

func prepopoulatePartyScores(t *testing.T, ctx context.Context, gs *gameScoresTestStore, now time.Time) []entities.GamePartyScore {
	t.Helper()
	team1 := entities.TeamID("AAAA")
	team2 := entities.TeamID("BBBB")
	team3 := entities.TeamID("CCCC")
	gps := []entities.GamePartyScore{
		{
			GameID:         "EEEE",
			EpochID:        1,
			PartyID:        "FFFE",
			TeamID:         &team3,
			Score:          num.DecimalFromFloat(0.1),
			StakingBalance: num.DecimalFromInt64(1),
			OpenVolume:     num.DecimalFromInt64(2),
			TotalFeesPaid:  num.DecimalFromInt64(3),
			IsEligible:     true,
			VegaTime:       now,
		},
		{
			GameID:         "FFFF",
			EpochID:        1,
			PartyID:        "FFFE",
			TeamID:         &team1,
			Score:          num.DecimalFromFloat(0.1),
			StakingBalance: num.DecimalFromInt64(1),
			OpenVolume:     num.DecimalFromInt64(2),
			TotalFeesPaid:  num.DecimalFromInt64(3),
			IsEligible:     true,
			VegaTime:       now,
		},
		{
			GameID:         "EEFF",
			EpochID:        1,
			PartyID:        "FFFD",
			TeamID:         &team2,
			Score:          num.DecimalFromFloat(0.2),
			StakingBalance: num.DecimalFromInt64(11),
			OpenVolume:     num.DecimalFromInt64(22),
			TotalFeesPaid:  num.DecimalFromInt64(33),
			IsEligible:     true,
			VegaTime:       now,
		},
		{
			GameID:         "FFFF",
			EpochID:        1,
			PartyID:        "FFFD",
			Score:          num.DecimalFromFloat(0.2),
			StakingBalance: num.DecimalFromInt64(111),
			OpenVolume:     num.DecimalFromInt64(222),
			TotalFeesPaid:  num.DecimalFromInt64(333),
			IsEligible:     true,
			VegaTime:       now,
		},
		{
			GameID:         "FFFF",
			EpochID:        1,
			PartyID:        "FFFC",
			TeamID:         &team3,
			Score:          num.DecimalFromFloat(0.3),
			StakingBalance: num.DecimalFromInt64(1111),
			OpenVolume:     num.DecimalFromInt64(2222),
			TotalFeesPaid:  num.DecimalFromInt64(3333),
			IsEligible:     true,
			VegaTime:       now,
		},
		{
			GameID:         "FFFF",
			EpochID:        1,
			PartyID:        "FFFB",
			TeamID:         &team3,
			Score:          num.DecimalFromFloat(0.4),
			StakingBalance: num.DecimalFromInt64(11111),
			OpenVolume:     num.DecimalFromInt64(22222),
			TotalFeesPaid:  num.DecimalFromInt64(33333),
			IsEligible:     true,
			VegaTime:       now,
		},
		{
			GameID:         "FFFF",
			EpochID:        1,
			PartyID:        "FFFA",
			Score:          num.DecimalFromFloat(0.5),
			StakingBalance: num.DecimalTwo(),
			OpenVolume:     num.DecimalZero(),
			TotalFeesPaid:  num.DecimalFromInt64(4),
			IsEligible:     true,
			VegaTime:       now,
		},
	}
	for _, gps1 := range gps {
		require.NoError(t, gs.gs.AddPartyScore(ctx, gps1))
	}
	sort.Slice(gps, func(i, j int) bool {
		if gps[i].GameID == gps[j].GameID {
			return gps[i].PartyID > gps[j].PartyID
		}
		return gps[i].GameID > gps[j].GameID
	})

	return gps
}

func TestListPartyScoresNoFilters(t *testing.T) {
	ctx := tempTransaction(t)
	store := newGameScoresTestStore(t)
	now := time.Now()
	pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil, true)
	partyScores := prepopoulatePartyScores(t, ctx, store, now)
	scores, _, err := store.gs.ListPartyScores(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)
	require.Equal(t, len(partyScores), len(scores))

	// now insert a fresh score for an existing party for the same game
	now = now.Add(time.Hour)
	partyScores[0].VegaTime = now
	partyScores[0].Score = num.DecimalE()
	require.NoError(t, store.gs.AddPartyScore(ctx, partyScores[0]))
	require.Equal(t, len(partyScores), len(scores))
}

func TestListPartyScoresPartyFilters(t *testing.T) {
	ctx := tempTransaction(t)
	store := newGameScoresTestStore(t)
	now := time.Now()
	pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil, true)
	prepopoulatePartyScores(t, ctx, store, now)
	scores, _, err := store.gs.ListPartyScores(ctx, nil, []entities.PartyID{"FFFD"}, nil, pagination)
	require.NoError(t, err)
	require.Equal(t, 2, len(scores))

	scores, _, err = store.gs.ListPartyScores(ctx, nil, []entities.PartyID{"FFFD", "FFFE"}, nil, pagination)
	require.NoError(t, err)
	require.Equal(t, 4, len(scores))
}

func TestListPartyScoresGameFilters(t *testing.T) {
	ctx := tempTransaction(t)
	store := newGameScoresTestStore(t)
	now := time.Now()
	pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil, true)
	ps := prepopoulatePartyScores(t, ctx, store, now)
	scores, _, err := store.gs.ListPartyScores(ctx, []entities.GameID{"EEFF"}, nil, nil, pagination)
	require.NoError(t, err)
	require.Equal(t, 1, len(scores))

	scores, _, err = store.gs.ListPartyScores(ctx, []entities.GameID{"FFFF", "EEEE"}, nil, nil, pagination)
	require.NoError(t, err)
	require.Equal(t, len(ps)-1, len(scores))
}

func TestListPartyScoresTeamFilters(t *testing.T) {
	ctx := tempTransaction(t)
	store := newGameScoresTestStore(t)
	now := time.Now()
	pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil, true)
	prepopoulatePartyScores(t, ctx, store, now)
	scores, _, err := store.gs.ListPartyScores(ctx, nil, nil, []entities.TeamID{"AAAA"}, pagination)
	require.NoError(t, err)
	require.Equal(t, 1, len(scores))
	scores, _, err = store.gs.ListPartyScores(ctx, nil, nil, []entities.TeamID{"AAAA", "BBBB"}, pagination)
	require.NoError(t, err)
	require.Equal(t, 2, len(scores))
}

func TestListPartyScoresAllFilters(t *testing.T) {
	ctx := tempTransaction(t)
	store := newGameScoresTestStore(t)
	now := time.Now()
	pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil, true)
	prepopoulatePartyScores(t, ctx, store, now)

	// all filters populated
	scores, _, err := store.gs.ListPartyScores(ctx, []entities.GameID{"FFFF"}, []entities.PartyID{"FFFB"}, []entities.TeamID{"CCCC"}, pagination)
	require.NoError(t, err)
	require.Equal(t, 1, len(scores))
	require.Equal(t, num.DecimalFromFloat(0.4), scores[0].Score)
	require.Equal(t, num.DecimalFromInt64(11111), scores[0].StakingBalance)
	require.Equal(t, num.DecimalFromInt64(22222), scores[0].OpenVolume)
	require.Equal(t, num.DecimalFromInt64(33333), scores[0].TotalFeesPaid)
}
