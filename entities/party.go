package entities

import (
	"time"
)

type PartyID struct{ ID }

func NewPartyID(id string) PartyID {
	return PartyID{ID: ID(id)}
}

type Party struct {
	ID       PartyID
	VegaTime time.Time
}
