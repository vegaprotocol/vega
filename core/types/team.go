package types

import (
	"time"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
)

const teamIDLength = 64

type TeamID string

type Team struct {
	ID TeamID

	Referrer *Membership
	Referees []*Membership

	Name      string
	TeamURL   string
	AvatarURL string
	CreatedAt time.Time
}

type Membership struct {
	PartyID       PartyID
	JoinedAt      time.Time
	NumberOfEpoch uint64
}

func (t *Team) AddReferee(partyID PartyID, joinedAt time.Time) {
	t.Referees = append(t.Referees, &Membership{
		PartyID:       partyID,
		JoinedAt:      joinedAt,
		NumberOfEpoch: 0,
	})
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

func NewTeamID() TeamID {
	return TeamID(vgrand.RandomStr(teamIDLength))
}
