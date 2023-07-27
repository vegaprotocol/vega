package types

import (
	vgrand "code.vegaprotocol.io/vega/libs/rand"
)

const teamIDLength = 64

type TeamID string

type Team struct {
	ID TeamID

	Referrer PartyID
	Referees []PartyID

	Name      string
	TeamURL   string
	AvatarURL string
}

func (t *Team) AddReferee(referee PartyID) {
	t.Referees = append(t.Referees, referee)
}

func (t *Team) RemoveReferee(refereeToRemove PartyID) {
	refereeIndex := 0
	for i, referee := range t.Referees {
		if referee == refereeToRemove {
			refereeIndex = i
			break
		}
	}

	lastIndex := len(t.Referees) - 1
	if refereeIndex < lastIndex {
		copy(t.Referees[refereeIndex:], t.Referees[refereeIndex+1:])
	}
	t.Referees[lastIndex] = ""
	t.Referees = t.Referees[:lastIndex]
}

func NewTeamID() TeamID {
	return TeamID(vgrand.RandomStr(teamIDLength))
}
