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

package gql

import (
	"context"
	"fmt"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type gameResolver VegaResolverRoot

func (g *gameResolver) Epoch(_ context.Context, obj *v2.Game) (int, error) {
	return int(obj.Epoch), nil
}

func (g *gameResolver) NumberOfParticipants(_ context.Context, obj *v2.Game) (int, error) {
	return int(obj.Participants), nil
}

func (g *gameResolver) Entities(_ context.Context, obj *v2.Game) ([]GameEntity, error) {
	switch e := obj.Entities.(type) {
	case *v2.Game_Team:
		return resolveTeamEntities(obj.GetTeam())
	case *v2.Game_Individual:
		return resolveIndividualEntities(obj.GetIndividual())
	default:
		return nil, fmt.Errorf("unsupported entity type: %T", e)
	}
}

func resolveTeamEntities(team *v2.TeamGameEntities) ([]GameEntity, error) {
	entities := make([]GameEntity, 0)
	for _, e := range team.Team {
		entity := TeamGameEntity{
			Team: &TeamParticipation{
				TeamID:               e.Team.TeamId,
				MembersParticipating: resolveIndividuals(e.Team.MembersParticipating),
			},
			Rank:                      int(e.Rank),
			Volume:                    e.Volume,
			RewardMetric:              e.RewardMetric,
			RewardEarned:              e.RewardEarned,
			TotalRewardsEarned:        e.TotalRewardsEarned,
			RewardEarnedQuantum:       e.RewardEarnedQuantum,
			TotalRewardsEarnedQuantum: e.TotalRewardsEarnedQuantum,
		}
		entities = append(entities, entity)
	}
	return entities, nil
}

func resolveIndividualEntities(individual *v2.IndividualGameEntities) ([]GameEntity, error) {
	entities := make([]GameEntity, 0)
	for _, e := range resolveIndividuals(individual.Individual) {
		entities = append(entities, e)
	}
	return entities, nil
}

func resolveIndividuals(individuals []*v2.IndividualGameEntity) []*IndividualGameEntity {
	entities := make([]*IndividualGameEntity, 0)
	for _, e := range individuals {
		entity := IndividualGameEntity{
			Individual:                e.Individual,
			Rank:                      int(e.Rank),
			Volume:                    e.Volume,
			RewardMetric:              e.RewardMetric,
			RewardEarned:              e.RewardEarned,
			TotalRewardsEarned:        e.TotalRewardsEarned,
			RewardEarnedQuantum:       e.RewardEarnedQuantum,
			TotalRewardsEarnedQuantum: e.TotalRewardsEarnedQuantum,
		}
		entities = append(entities, &entity)
	}
	return entities
}
