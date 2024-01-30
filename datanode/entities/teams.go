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
)

type (
	_Team  struct{}
	TeamID = ID[_Team]
)

type Team struct {
	ID             TeamID
	Referrer       PartyID
	Name           string
	TeamURL        *string
	AvatarURL      *string
	Closed         bool
	AllowList      []string
	TotalMembers   uint64
	CreatedAt      time.Time
	CreatedAtEpoch uint64
	VegaTime       time.Time
}

func (t Team) Cursor() *Cursor {
	tc := TeamCursor{
		CreatedAt: t.CreatedAt,
		ID:        t.ID,
	}
	return NewCursor(tc.String())
}

func (t Team) ToProto() *v2.Team {
	return &v2.Team{
		TeamId:         string(t.ID),
		Referrer:       string(t.Referrer),
		Name:           t.Name,
		TeamUrl:        t.TeamURL,
		AvatarUrl:      t.AvatarURL,
		CreatedAt:      t.CreatedAt.UnixNano(),
		Closed:         t.Closed,
		AllowList:      t.AllowList,
		CreatedAtEpoch: t.CreatedAtEpoch,
		TotalMembers:   t.TotalMembers,
	}
}

func (t Team) ToProtoEdge(_ ...any) (*v2.TeamEdge, error) {
	return &v2.TeamEdge{
		Node:   t.ToProto(),
		Cursor: t.Cursor().Encode(),
	}, nil
}

func TeamCreatedFromProto(created *eventspb.TeamCreated, vegaTime time.Time) *Team {
	return &Team{
		ID:             TeamID(created.TeamId),
		Referrer:       PartyID(created.Referrer),
		Name:           created.Name,
		TeamURL:        created.TeamUrl,
		AvatarURL:      created.AvatarUrl,
		CreatedAt:      time.Unix(0, created.CreatedAt),
		CreatedAtEpoch: created.AtEpoch,
		VegaTime:       vegaTime,
		Closed:         created.Closed,
		AllowList:      created.AllowList,
	}
}

type TeamUpdated struct {
	ID        TeamID
	Name      string
	TeamURL   *string
	AvatarURL *string
	Closed    bool
	AllowList []string
	VegaTime  time.Time
}

func TeamUpdatedFromProto(updated *eventspb.TeamUpdated, vegaTime time.Time) *TeamUpdated {
	return &TeamUpdated{
		ID:        TeamID(updated.TeamId),
		Name:      updated.Name,
		TeamURL:   updated.TeamUrl,
		AvatarURL: updated.AvatarUrl,
		Closed:    updated.Closed,
		AllowList: updated.AllowList,
		VegaTime:  vegaTime,
	}
}

type TeamCursor struct {
	CreatedAt time.Time
	ID        TeamID
}

func (tc TeamCursor) String() string {
	bs, err := json.Marshal(tc)
	if err != nil {
		panic(fmt.Errorf("could not marshal team cursor: %v", err))
	}
	return string(bs)
}

func (tc *TeamCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), tc)
}

type TeamsStatistics struct {
	TeamID              TeamID
	TotalQuantumRewards num.Decimal
	QuantumRewards      []QuantumRewardsPerEpoch
	TotalGamesPlayed    uint64
	GamesPlayed         []GameID
}

type QuantumRewardsPerEpoch struct {
	Epoch uint64
	Total num.Decimal
}

func (t TeamsStatistics) Cursor() *Cursor {
	tc := TeamsStatisticsCursor{
		ID: t.TeamID,
	}
	return NewCursor(tc.String())
}

func (t TeamsStatistics) ToProto() *v2.TeamStatistics {
	gamesPlayed := make([]string, 0, len(t.GamesPlayed))
	for _, id := range t.GamesPlayed {
		gamesPlayed = append(gamesPlayed, id.String())
	}

	quantumRewards := make([]*v2.QuantumRewardsPerEpoch, 0, len(t.QuantumRewards))
	for _, r := range t.QuantumRewards {
		quantumRewards = append(quantumRewards, &v2.QuantumRewardsPerEpoch{
			Epoch:               r.Epoch,
			TotalQuantumRewards: r.Total.String(),
		})
	}

	return &v2.TeamStatistics{
		TeamId:              string(t.TeamID),
		TotalQuantumVolume:  "",
		TotalQuantumRewards: t.TotalQuantumRewards.String(),
		QuantumRewards:      quantumRewards,
		TotalGamesPlayed:    t.TotalGamesPlayed,
		GamesPlayed:         gamesPlayed,
	}
}

func (t TeamsStatistics) ToProtoEdge(_ ...any) (*v2.TeamStatisticsEdge, error) {
	return &v2.TeamStatisticsEdge{
		Node:   t.ToProto(),
		Cursor: t.Cursor().Encode(),
	}, nil
}

type TeamsStatisticsCursor struct {
	ID TeamID
}

func (c TeamsStatisticsCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal teams stats cursor: %v", err))
	}
	return string(bs)
}

func (c *TeamsStatisticsCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}

type TeamMembersStatistics struct {
	PartyID             PartyID
	TotalQuantumRewards num.Decimal
	QuantumRewards      []QuantumRewardsPerEpoch
	TotalGamesPlayed    uint64
	GamesPlayed         []GameID
}

func (t TeamMembersStatistics) Cursor() *Cursor {
	tc := TeamMemberStatisticsCursor{
		ID: t.PartyID,
	}
	return NewCursor(tc.String())
}

func (t TeamMembersStatistics) ToProto() *v2.TeamMemberStatistics {
	gamesPlayed := make([]string, 0, len(t.GamesPlayed))
	for _, id := range t.GamesPlayed {
		gamesPlayed = append(gamesPlayed, id.String())
	}

	quantumRewards := make([]*v2.QuantumRewardsPerEpoch, 0, len(t.QuantumRewards))
	for _, r := range t.QuantumRewards {
		quantumRewards = append(quantumRewards, &v2.QuantumRewardsPerEpoch{
			Epoch:               r.Epoch,
			TotalQuantumRewards: r.Total.String(),
		})
	}

	return &v2.TeamMemberStatistics{
		PartyId:             string(t.PartyID),
		TotalQuantumVolume:  "",
		TotalQuantumRewards: t.TotalQuantumRewards.String(),
		QuantumRewards:      quantumRewards,
		TotalGamesPlayed:    t.TotalGamesPlayed,
		GamesPlayed:         gamesPlayed,
	}
}

func (t TeamMembersStatistics) ToProtoEdge(_ ...any) (*v2.TeamMemberStatisticsEdge, error) {
	return &v2.TeamMemberStatisticsEdge{
		Node:   t.ToProto(),
		Cursor: t.Cursor().Encode(),
	}, nil
}

type TeamMemberStatisticsCursor struct {
	ID PartyID
}

func (c TeamMemberStatisticsCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal team member stats cursor: %v", err))
	}
	return string(bs)
}

func (c *TeamMemberStatisticsCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}

type TeamMember struct {
	TeamID        TeamID
	PartyID       PartyID
	JoinedAt      time.Time
	JoinedAtEpoch uint64
	VegaTime      time.Time
}

func (t TeamMember) Cursor() *Cursor {
	rc := RefereeCursor{
		PartyID: t.PartyID,
	}
	return NewCursor(rc.String())
}

func (t TeamMember) ToProto() *v2.TeamReferee {
	return &v2.TeamReferee{
		TeamId:        string(t.TeamID),
		Referee:       string(t.PartyID),
		JoinedAt:      t.JoinedAt.UnixNano(),
		JoinedAtEpoch: t.JoinedAtEpoch,
	}
}

func (t TeamMember) ToProtoEdge(_ ...any) (*v2.TeamRefereeEdge, error) {
	return &v2.TeamRefereeEdge{
		Node:   t.ToProto(),
		Cursor: t.Cursor().Encode(),
	}, nil
}

func TeamRefereeFromProto(joined *eventspb.RefereeJoinedTeam, vegaTime time.Time) *TeamMember {
	return &TeamMember{
		TeamID:        TeamID(joined.TeamId),
		PartyID:       PartyID(joined.Referee),
		JoinedAt:      time.Unix(0, joined.JoinedAt),
		JoinedAtEpoch: joined.AtEpoch,
		VegaTime:      vegaTime,
	}
}

type RefereeCursor struct {
	PartyID PartyID
}

func (rc RefereeCursor) String() string {
	bs, err := json.Marshal(rc)
	if err != nil {
		panic(fmt.Errorf("could not marshal referee cursor: %v", err))
	}
	return string(bs)
}

func (rc *RefereeCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), rc)
}

type TeamMemberHistory struct {
	TeamID        TeamID
	JoinedAt      time.Time
	JoinedAtEpoch uint64
}

func (t TeamMemberHistory) Cursor() *Cursor {
	rc := RefereeHistoryCursor{
		JoinedAtEpoch: t.JoinedAtEpoch,
	}
	return NewCursor(rc.String())
}

func (t TeamMemberHistory) ToProto() *v2.TeamRefereeHistory {
	return &v2.TeamRefereeHistory{
		TeamId:        string(t.TeamID),
		JoinedAt:      t.JoinedAt.UnixNano(),
		JoinedAtEpoch: t.JoinedAtEpoch,
	}
}

func (t TeamMemberHistory) ToProtoEdge(_ ...any) (*v2.TeamRefereeHistoryEdge, error) {
	return &v2.TeamRefereeHistoryEdge{
		Node: &v2.TeamRefereeHistory{
			TeamId:        string(t.TeamID),
			JoinedAt:      t.JoinedAt.UnixNano(),
			JoinedAtEpoch: t.JoinedAtEpoch,
		},
		Cursor: t.Cursor().Encode(),
	}, nil
}

type RefereeHistoryCursor struct {
	JoinedAtEpoch uint64
}

func (rh RefereeHistoryCursor) String() string {
	bs, err := json.Marshal(rh)
	if err != nil {
		panic(fmt.Errorf("could not marshal referee history cursor: %v", err))
	}
	return string(bs)
}

func (rh *RefereeHistoryCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), rh)
}

type RefereeTeamSwitch struct {
	FromTeamID      TeamID
	ToTeamID        TeamID
	PartyID         PartyID
	SwitchedAt      time.Time
	SwitchedAtEpoch uint64
	VegaTime        time.Time
}

func TeamRefereeHistoryFromProto(switched *eventspb.RefereeSwitchedTeam, vegaTime time.Time) *RefereeTeamSwitch {
	return &RefereeTeamSwitch{
		FromTeamID:      TeamID(switched.FromTeamId),
		ToTeamID:        TeamID(switched.ToTeamId),
		PartyID:         PartyID(switched.Referee),
		SwitchedAt:      time.Unix(0, switched.SwitchedAt),
		SwitchedAtEpoch: switched.AtEpoch,
		VegaTime:        vegaTime,
	}
}
