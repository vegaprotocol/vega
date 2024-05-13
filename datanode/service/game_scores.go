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

package service

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
)

type gameScoreStore interface {
	ListPartyScores(
		ctx context.Context,
		gameIDs []entities.GameID,
		partyIDs []entities.PartyID,
		teamIDs []entities.TeamID,
		pagination entities.CursorPagination,
	) ([]entities.GamePartyScore, entities.PageInfo, error)
	ListTeamScores(
		ctx context.Context,
		gameIDs []entities.GameID,
		teamIDs []entities.TeamID,
		pagination entities.CursorPagination,
	) ([]entities.GameTeamScore, entities.PageInfo, error)
}

type GameScore struct {
	store gameScoreStore
}

func NewGameScore(store gameScoreStore, log *logging.Logger) *GameScore {
	return &GameScore{
		store: store,
	}
}

func (gs *GameScore) ListPartyScores(
	ctx context.Context,
	gameIDs []entities.GameID,
	partyIDs []entities.PartyID,
	teamIDs []entities.TeamID,
	pagination entities.CursorPagination,
) ([]entities.GamePartyScore, entities.PageInfo, error) {
	return gs.store.ListPartyScores(ctx, gameIDs, partyIDs, teamIDs, pagination)
}

func (gs *GameScore) ListTeamScores(
	ctx context.Context,
	gameIDs []entities.GameID,
	teamIDs []entities.TeamID,
	pagination entities.CursorPagination,
) ([]entities.GameTeamScore, entities.PageInfo, error) {
	return gs.store.ListTeamScores(ctx, gameIDs, teamIDs, pagination)
}
