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
