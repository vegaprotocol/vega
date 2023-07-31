package teams

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
)

func ErrNoTeamMatchesID(id types.TeamID) error {
	return fmt.Errorf("no team matches ID %q", id)
}

func ErrPartyAlreadyBelongsToTeam(referrer types.PartyID) error {
	return fmt.Errorf("the party %q already belongs to a team", referrer)
}
