package entities

import (
	"time"

	"code.vegaprotocol.io/vega/types"
)

type PartyID struct{ ID }

func NewPartyID(id string) PartyID {
	return PartyID{ID: ID(id)}
}

type Party struct {
	ID       PartyID
	VegaTime *time.Time // Can be NULL for built-in party 'network'
}

func PartyFromProto(pp *types.Party) Party {
	return Party{ID: NewPartyID(pp.Id)}
}

func (p *Party) ToProto() *types.Party {
	return &types.Party{Id: p.ID.String()}
}
