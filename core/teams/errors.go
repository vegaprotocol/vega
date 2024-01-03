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

package teams

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
)

var (
	ErrOnlyReferrerCanUpdateTeam     = errors.New("only the referrer can update the team properties")
	ErrReferrerCannotJoinAnotherTeam = errors.New("a referrer cannot join another team")
	ErrComputedTeamIDIsAlreadyInUse  = errors.New("the computed team ID is already in use")
	ErrTeamNameIsAlreadyInUse        = errors.New("the team name is already in use")
)

func ErrNoTeamMatchesID(id types.TeamID) error {
	return fmt.Errorf("no team matches ID %q", id)
}

func ErrPartyAlreadyBelongsToTeam(referrer types.PartyID) error {
	return fmt.Errorf("the party %q already belongs to a team", referrer)
}
