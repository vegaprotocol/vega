package entities

import (
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
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

func (p Party) Cursor() *Cursor {
	return NewCursor(p.VegaTime.In(time.UTC).Format(time.RFC3339Nano))
}

func (p Party) ToProtoEdge(_ ...any) *v2.PartyEdge {
	return &v2.PartyEdge{
		Node:   p.ToProto(),
		Cursor: p.Cursor().Encode(),
	}
}
