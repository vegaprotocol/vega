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
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/shopspring/decimal"
)

type GamePartyScore struct {
	GameID         GameID
	TeamID         *TeamID
	EpochID        int64
	PartyID        PartyID
	Score          decimal.Decimal
	StakingBalance decimal.Decimal
	OpenVolume     decimal.Decimal
	TotalFeesPaid  decimal.Decimal
	IsEligible     bool
	Rank           *uint64
	VegaTime       time.Time
}

func (pgs GamePartyScore) ToProto() *eventspb.GamePartyScore {
	var teamID *string
	if pgs.TeamID != nil {
		tid := pgs.TeamID.String()
		teamID = &tid
	}
	return &eventspb.GamePartyScore{
		GameId:         pgs.GameID.String(),
		Party:          pgs.PartyID.String(),
		Epoch:          pgs.EpochID,
		TeamId:         teamID,
		Score:          pgs.Score.String(),
		StakingBalance: pgs.StakingBalance.String(),
		OpenVolume:     pgs.OpenVolume.String(),
		TotalFeesPaid:  pgs.TotalFeesPaid.String(),
		IsEligible:     pgs.IsEligible,
		Rank:           pgs.Rank,
		Time:           pgs.VegaTime.UnixNano(),
	}
}

type GameTeamScore struct {
	GameID   GameID
	TeamID   TeamID
	EpochID  int64
	Score    decimal.Decimal
	VegaTime time.Time
}

func (pgs GameTeamScore) ToProto() *eventspb.GameTeamScore {
	return &eventspb.GameTeamScore{
		GameId: pgs.GameID.String(),
		Epoch:  pgs.EpochID,
		TeamId: pgs.TeamID.String(),
		Score:  pgs.Score.String(),
		Time:   pgs.VegaTime.UnixNano(),
	}
}

func GameScoresFromProto(gs *eventspb.GameScores, txHash TxHash, vegaTime time.Time, seqNum uint64) ([]GameTeamScore, []GamePartyScore, error) {
	ts := make([]GameTeamScore, 0, len(gs.TeamScores))
	ps := []GamePartyScore{}
	for _, gsTeam := range gs.TeamScores {
		score, err := num.DecimalFromString(gsTeam.Score)
		if err != nil {
			return nil, nil, err
		}
		ts = append(ts, GameTeamScore{
			GameID:   (ID[_Game])(gsTeam.GameId),
			TeamID:   ID[_Team](gsTeam.TeamId),
			EpochID:  gsTeam.Epoch,
			Score:    score,
			VegaTime: vegaTime,
		})
	}
	for _, gsParty := range gs.PartyScores {
		score, err := num.DecimalFromString(gsParty.Score)
		if err != nil {
			return nil, nil, err
		}
		var stakingBalance num.Decimal
		if len(gsParty.StakingBalance) > 0 {
			stakingBalance, err = num.DecimalFromString(gsParty.StakingBalance)
			if err != nil {
				return nil, nil, err
			}
		}
		var openVolume num.Decimal
		if len(gsParty.OpenVolume) > 0 {
			openVolume, err = num.DecimalFromString(gsParty.OpenVolume)
			if err != nil {
				return nil, nil, err
			}
		}
		var totalFeesPaid num.Decimal
		if len(gsParty.TotalFeesPaid) > 0 {
			totalFeesPaid, err = num.DecimalFromString(gsParty.TotalFeesPaid)
			if err != nil {
				return nil, nil, err
			}
		}
		ps = append(ps, GamePartyScore{
			GameID:         (ID[_Game])(gsParty.GameId),
			EpochID:        gsParty.Epoch,
			PartyID:        ID[_Party](gsParty.Party),
			Score:          score,
			StakingBalance: stakingBalance,
			OpenVolume:     openVolume,
			TotalFeesPaid:  totalFeesPaid,
			IsEligible:     gsParty.IsEligible,
			VegaTime:       vegaTime,
		})
	}

	return ts, ps, nil
}

func (pgs GamePartyScore) Cursor() *Cursor {
	cursor := PartyGameScoreCursor{
		GameID:   pgs.GameID.String(),
		PartyID:  pgs.PartyID.String(),
		EpochID:  pgs.EpochID,
		VegaTime: pgs.VegaTime,
	}
	return NewCursor(cursor.String())
}

func (pgs GamePartyScore) ToProtoEdge(_ ...any) (*v2.GamePartyScoresEdge, error) {
	return &v2.GamePartyScoresEdge{
		Node:   pgs.ToProto(),
		Cursor: pgs.Cursor().Encode(),
	}, nil
}

func (pgs GameTeamScore) Cursor() *Cursor {
	cursor := TeamGameScoreCursor{
		GameID:   pgs.GameID.String(),
		TeamID:   pgs.TeamID.String(),
		EpochID:  pgs.EpochID,
		VegaTime: pgs.VegaTime,
	}
	return NewCursor(cursor.String())
}

func (pgs GameTeamScore) ToProtoEdge(_ ...any) (*v2.GameTeamScoresEdge, error) {
	return &v2.GameTeamScoresEdge{
		Node:   pgs.ToProto(),
		Cursor: pgs.Cursor().Encode(),
	}, nil
}

type PartyGameScoreCursor struct {
	GameID   string    `json:"game_id"`
	PartyID  string    `json:"party_id"`
	EpochID  int64     `json:"epoch_id"`
	VegaTime time.Time `json:"vega_time"`
}

func (pg PartyGameScoreCursor) String() string {
	bs, err := json.Marshal(pg)
	if err != nil {
		panic(fmt.Errorf("marshalling party game cursor: %w", err))
	}
	return string(bs)
}

func (pg *PartyGameScoreCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), pg)
}

type TeamGameScoreCursor struct {
	GameID   string    `json:"game_id"`
	TeamID   string    `json:"team_id"`
	EpochID  int64     `json:"epoch_id"`
	VegaTime time.Time `json:"vega_time"`
}

func (pg TeamGameScoreCursor) String() string {
	bs, err := json.Marshal(pg)
	if err != nil {
		panic(fmt.Errorf("marshalling team game score cursor: %w", err))
	}
	return string(bs)
}

func (pg *TeamGameScoreCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), pg)
}
