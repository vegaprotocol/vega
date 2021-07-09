package events

import (
	"context"

	ptypes "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	"code.vegaprotocol.io/vega/types"
)

type Acc struct {
	*Base
	a ptypes.Account
}

func NewAccountEvent(ctx context.Context, a types.Account) *Acc {
	return &Acc{
		Base: newBase(ctx, AccountEvent),
		a:    *(a.IntoProto()),
	}
}

func (a Acc) IsParty(id string) bool {
	return a.a.Owner == id
}

func (a Acc) PartyID() string {
	return a.a.Owner
}

func (a Acc) MarketID() string {
	return a.a.MarketId
}

func (a *Acc) Account() ptypes.Account {
	return a.a
}

func (a Acc) Proto() ptypes.Account {
	return a.a
}

func (a Acc) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    a.eventID(),
		Block: a.TraceID(),
		Type:  a.et.ToProto(),
		Event: &eventspb.BusEvent_Account{
			Account: &a.a,
		},
	}
}
