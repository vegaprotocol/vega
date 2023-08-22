// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
)

func ErrNoTeamMatchesID(id types.TeamID) error {
	return fmt.Errorf("no team matches ID %q", id)
}

func ErrPartyAlreadyBelongsToTeam(referrer types.PartyID) error {
	return fmt.Errorf("the party %q already belongs to a team", referrer)
}
