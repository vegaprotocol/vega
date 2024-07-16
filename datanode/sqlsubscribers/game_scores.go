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

package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
)

type GameScoreEvent interface {
	events.Event
	GameScoreEvent() eventspb.GameScores
}

type GameScoreStore interface {
	AddTeamScore(context.Context, entities.GameTeamScore) error
	AddPartyScore(context.Context, entities.GamePartyScore) error
}

type GameScore struct {
	subscriber
	store GameScoreStore
}

func NewGameScore(store GameScoreStore) *GameScore {
	t := &GameScore{
		store: store,
	}
	return t
}

func (gs *GameScore) Types() []events.Type {
	return []events.Type{events.GameScoresEvent}
}

func (gs *GameScore) Push(ctx context.Context, evt events.Event) error {
	return gs.consume(ctx, evt.(GameScoreEvent))
}

func (gs *GameScore) consume(ctx context.Context, event GameScoreEvent) error {
	gameScoresEvents := event.GameScoreEvent()
	teamScores, partyScores, err := entities.GameScoresFromProto(&gameScoresEvents, entities.TxHash(event.TxHash()), gs.vegaTime, event.Sequence())
	if err != nil {
		return errors.Wrap(err, "unable to parse game scores")
	}

	for _, ts := range teamScores {
		if err := gs.store.AddTeamScore(ctx, ts); err != nil {
			return errors.Wrap(err, "error adding team score")
		}
	}
	for _, ps := range partyScores {
		if err := gs.store.AddPartyScore(ctx, ps); err != nil {
			return errors.Wrap(err, "error adding party score")
		}
	}
	return nil
}

func (gs *GameScore) Name() string {
	return "GameScore"
}
