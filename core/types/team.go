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

package types

import (
	"fmt"
	"time"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
)

const teamIDLength = 64

type TeamID string

func (t TeamID) IsNoTeam() bool {
	return len(string(t)) <= 0
}

type Team struct {
	ID TeamID

	Referrer *Membership
	Referees []*Membership

	Name      string
	TeamURL   string
	AvatarURL string
	CreatedAt time.Time

	Closed    bool
	AllowList []PartyID
}

type Membership struct {
	PartyID        PartyID
	JoinedAt       time.Time
	StartedAtEpoch uint64
}

func (t *Team) RemoveReferee(refereeToRemove PartyID) {
	refereeIndex := 0
	for i, referee := range t.Referees {
		if referee.PartyID == refereeToRemove {
			refereeIndex = i
			break
		}
	}

	lastIndex := len(t.Referees) - 1
	if refereeIndex < lastIndex {
		copy(t.Referees[refereeIndex:], t.Referees[refereeIndex+1:])
	}
	t.Referees[lastIndex] = nil
	t.Referees = t.Referees[:lastIndex]
}

func (t *Team) EnsureCanJoin(party PartyID) error {
	if !t.Closed {
		return nil
	}

	if len(t.AllowList) == 0 {
		return ErrTeamIsClosed(t.ID)
	}

	for _, allowedParty := range t.AllowList {
		if allowedParty == party {
			return nil
		}
	}

	return ErrRefereeNotAllowedToJoinTeam(t.ID)
}

func NewTeamID() TeamID {
	return TeamID(vgrand.RandomStr(teamIDLength))
}

func ErrTeamIsClosed(id TeamID) error {
	return fmt.Errorf("team %q is closed", id)
}

func ErrRefereeNotAllowedToJoinTeam(id TeamID) error {
	return fmt.Errorf("party is not allowed to join team %q", id)
}
