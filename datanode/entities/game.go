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

package entities

import (
	"encoding/json"
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type _Game struct{}

type GameID = ID[_Game]

type Game struct {
	ID           GameID
	Epoch        uint64
	Participants uint64
	Entities     []GameEntity
}

func (g Game) Cursor() *Cursor {
	gc := GameCursor{
		EpochID: g.Epoch,
		GameID:  g.ID,
	}
	return NewCursor(gc.String())
}

func (g Game) ToProtoEdge(_ ...any) (*v2.GameEdge, error) {
	return &v2.GameEdge{
		Node:   g.ToProto(),
		Cursor: g.Cursor().Encode(),
	}, nil
}

type GameCursor struct {
	EpochID uint64
	GameID  GameID
}

func (gc GameCursor) String() string {
	bs, err := json.Marshal(gc)
	if err != nil {
		panic(fmt.Errorf("could not marshal game cursor %v", err))
	}
	return string(bs)
}

func (gc *GameCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), gc)
}

type GameEntity interface {
	IsGameEntity()
}

type TeamGameParticipation struct {
	TeamID               TeamID
	MembersParticipating []IndividualGameEntity
}

func (t TeamGameParticipation) ToProto() *v2.TeamGameParticipation {
	members := make([]*v2.IndividualGameEntity, len(t.MembersParticipating))
	for i, member := range t.MembersParticipating {
		members[i] = member.ToProto()
	}
	return &v2.TeamGameParticipation{
		TeamId:               t.TeamID.String(),
		MembersParticipating: members,
	}
}

type TeamGameEntity struct {
	Team               TeamGameParticipation
	Rank               uint64
	Volume             num.Decimal
	RewardMetric       string
	RewardEarned       *num.Uint
	TotalRewardsEarned *num.Uint
}

func (*TeamGameEntity) IsGameEntity() {}
func (t *TeamGameEntity) ToProto() *v2.TeamGameEntity {
	return &v2.TeamGameEntity{
		Team:               t.Team.ToProto(),
		Rank:               t.Rank,
		Volume:             t.Volume.String(),
		RewardMetric:       t.RewardMetric,
		RewardEarned:       t.RewardEarned.String(),
		TotalRewardsEarned: t.TotalRewardsEarned.String(),
	}
}

type IndividualGameEntity struct {
	Individual         string
	Rank               uint64
	Volume             num.Decimal
	RewardMetric       string
	RewardEarned       *num.Uint
	TotalRewardsEarned *num.Uint
}

func (*IndividualGameEntity) IsGameEntity() {}

func (i *IndividualGameEntity) ToProto() *v2.IndividualGameEntity {
	return &v2.IndividualGameEntity{
		Individual:         i.Individual,
		Rank:               i.Rank,
		Volume:             i.Volume.String(),
		RewardMetric:       i.RewardMetric,
		RewardEarned:       i.RewardEarned.String(),
		TotalRewardsEarned: i.TotalRewardsEarned.String(),
	}
}

func (g Game) ToProto() *v2.Game {
	gg := &v2.Game{
		Id:           g.ID.String(),
		Epoch:        g.Epoch,
		Participants: g.Participants,
		Entities:     nil,
	}
	teamEntities := make([]*v2.TeamGameEntity, 0)
	individualEntities := make([]*v2.IndividualGameEntity, 0)
	for _, e := range g.Entities {
		switch entity := e.(type) {
		case *TeamGameEntity:
			teamEntities = append(teamEntities, entity.ToProto())
		case *IndividualGameEntity:
			individualEntities = append(individualEntities, entity.ToProto())
		}
	}
	if len(teamEntities) > 0 {
		gg.Entities = &v2.Game_Team{
			Team: &v2.TeamGameEntities{
				Team: teamEntities,
			},
		}
		return gg
	}
	gg.Entities = &v2.Game_Individual{
		Individual: &v2.IndividualGameEntities{
			Individual: individualEntities,
		},
	}
	return gg
}
