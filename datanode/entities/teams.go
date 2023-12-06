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

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	_Team  struct{}
	TeamID = ID[_Team]

	Team struct {
		ID             TeamID
		Referrer       PartyID
		Name           string
		TeamURL        *string
		AvatarURL      *string
		Closed         bool
		CreatedAt      time.Time
		CreatedAtEpoch uint64
		VegaTime       time.Time
	}

	TeamUpdated struct {
		ID        TeamID
		Name      string
		TeamURL   *string
		AvatarURL *string
		Closed    bool
		VegaTime  time.Time
	}

	TeamCursor struct {
		CreatedAt time.Time
		ID        TeamID
	}

	TeamMember struct {
		TeamID        TeamID
		PartyID       PartyID
		JoinedAt      time.Time
		JoinedAtEpoch uint64
		VegaTime      time.Time
	}

	TeamMemberHistory struct {
		TeamID        TeamID
		JoinedAt      time.Time
		JoinedAtEpoch uint64
	}

	RefereeTeamSwitch struct {
		FromTeamID      TeamID
		ToTeamID        TeamID
		PartyID         PartyID
		SwitchedAt      time.Time
		SwitchedAtEpoch uint64
		VegaTime        time.Time
	}

	RefereeCursor struct {
		PartyID PartyID
	}

	RefereeHistoryCursor struct {
		JoinedAtEpoch uint64
	}
)

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
	}
}

func TeamUpdatedFromProto(updated *eventspb.TeamUpdated, vegaTime time.Time) *TeamUpdated {
	return &TeamUpdated{
		ID:        TeamID(updated.TeamId),
		Name:      updated.Name,
		TeamURL:   updated.TeamUrl,
		AvatarURL: updated.AvatarUrl,
		Closed:    updated.Closed,
		VegaTime:  vegaTime,
	}
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
		CreatedAtEpoch: t.CreatedAtEpoch,
	}
}

func (t Team) ToProtoEdge(_ ...any) (*v2.TeamEdge, error) {
	return &v2.TeamEdge{
		Node:   t.ToProto(),
		Cursor: t.Cursor().Encode(),
	}, nil
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
