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

package events

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type GameScores struct {
	*Base
	pb eventspb.GameScores
}

func (gs *GameScores) GameScoreEvent() eventspb.GameScores {
	return gs.pb
}

func NewPartyGameScoresEvent(ctx context.Context, epoch int64, gameID string, time time.Time, partyScores []*types.PartyContributionScore) *GameScores {
	ps := make([]*eventspb.GamePartyScore, 0, len(partyScores))
	for _, partyScore := range partyScores {
		var ov, sb, tfp string
		if partyScore.OpenVolume != nil {
			ov = partyScore.OpenVolume.String()
		}
		if partyScore.StakingBalance != nil {
			sb = partyScore.StakingBalance.String()
		}
		if partyScore.TotalFeesPaid != nil {
			tfp = partyScore.TotalFeesPaid.String()
		}

		ps = append(ps, &eventspb.GamePartyScore{
			GameId:         gameID,
			Epoch:          epoch,
			Time:           time.UnixNano(),
			Party:          partyScore.Party,
			Score:          partyScore.Score.String(),
			IsEligible:     partyScore.IsEligible,
			OpenVolume:     ov,
			StakingBalance: sb,
			TotalFeesPaid:  tfp,
		})
	}

	return &GameScores{
		Base: newBase(ctx, GameScoresEvent),
		pb: eventspb.GameScores{
			PartyScores: ps,
		},
	}
}

func NewTeamGameScoresEvent(ctx context.Context, epoch int64, gameID string, time time.Time, teamScores []*types.PartyContributionScore, teamPartyScores map[string][]*types.PartyContributionScore) *GameScores {
	ts := make([]*eventspb.GameTeamScore, 0, len(teamScores))
	ps := []*eventspb.GamePartyScore{}
	for _, teamScore := range teamScores {
		team := &eventspb.GameTeamScore{
			GameId: gameID,
			Time:   time.UnixNano(),
			Epoch:  epoch,
			TeamId: teamScore.Party,
			Score:  teamScore.Score.String(),
		}
		for _, partyScore := range teamPartyScores[teamScore.Party] {
			var rank *uint64
			if partyScore.RankingIndex >= 0 {
				r := uint64(partyScore.RankingIndex)
				rank = &r
			}
			var ov, sb, tfp string
			if partyScore.OpenVolume != nil {
				ov = partyScore.OpenVolume.String()
			}
			if partyScore.StakingBalance != nil {
				sb = partyScore.StakingBalance.String()
			}
			if partyScore.TotalFeesPaid != nil {
				tfp = partyScore.TotalFeesPaid.String()
			}
			ps = append(ps, &eventspb.GamePartyScore{
				GameId:         gameID,
				TeamId:         &team.TeamId,
				Time:           time.UnixNano(),
				Epoch:          epoch,
				Party:          partyScore.Party,
				Score:          partyScore.Score.String(),
				IsEligible:     partyScore.IsEligible,
				OpenVolume:     ov,
				StakingBalance: sb,
				TotalFeesPaid:  tfp,
				Rank:           rank,
			})
		}
		ts = append(ts, team)
	}

	return &GameScores{
		Base: newBase(ctx, GameScoresEvent),
		pb: eventspb.GameScores{
			PartyScores: ps,
			TeamScores:  ts,
		},
	}
}

func (gs *GameScores) Proto() eventspb.GameScores {
	return gs.pb
}

func (gs *GameScores) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(gs.Base)
	cpy := gs.pb
	busEvent.Event = &eventspb.BusEvent_GameScores{
		GameScores: &cpy,
	}

	return busEvent
}

func GameScoresEventFromStream(ctx context.Context, be *eventspb.BusEvent) *GameScores {
	m := be.GetGameScores()
	return &GameScores{
		Base: newBaseFromBusEvent(ctx, GameScoresEvent, be),
		pb:   *m,
	}
}
