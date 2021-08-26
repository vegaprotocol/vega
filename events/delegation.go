package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type DelegationBalance struct {
	*Base
	party    string
	nodeID   string
	amount   *num.Uint
	epochSeq string
}

func NewDelegationBalance(ctx context.Context, party, nodeID string, amount *num.Uint, epochSeq string) *DelegationBalance {
	return &DelegationBalance{
		Base:     newBase(ctx, DelegationBalanceEvent),
		party:    party,
		nodeID:   nodeID,
		amount:   amount,
		epochSeq: epochSeq,
	}
}

func (db DelegationBalance) Proto() eventspb.DelegationBalanceEvent {
	return eventspb.DelegationBalanceEvent{
		Party:    db.party,
		NodeId:   db.nodeID,
		Amount:   db.amount.Uint64(),
		EpochSeq: db.epochSeq,
	}
}

func (db DelegationBalance) StreamMessage() *eventspb.BusEvent {
	p := db.Proto()
	return &eventspb.BusEvent{
		Id:    db.eventID(),
		Block: db.TraceID(),
		Type:  db.et.ToProto(),
		Event: &eventspb.BusEvent_DelegationBalance{
			DelegationBalance: &p,
		},
	}
}

func DelegationBalanceEventFromStream(ctx context.Context, be *eventspb.BusEvent) *DelegationBalance {
	event := be.GetDelegationBalance()
	if event == nil {
		return nil
	}

	return &DelegationBalance{
		Base:     newBaseFromStream(ctx, DelegationBalanceEvent, be),
		party:    event.GetParty(),
		nodeID:   event.GetNodeId(),
		amount:   num.NewUint(event.GetAmount()),
		epochSeq: event.GetEpochSeq(),
	}
}
