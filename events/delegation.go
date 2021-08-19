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

type PendingDelegationBalance struct {
	*Base
	party              string
	nodeID             string
	delegationAmount   *num.Uint
	undelegationAmount *num.Uint
	epochSeq           string
}

func NewPendingDelegationBalance(ctx context.Context, party, nodeID string, delegationAmount *num.Uint, undelegationAmount *num.Uint, epochSeq string) *PendingDelegationBalance {
	return &PendingDelegationBalance{
		Base:               newBase(ctx, PendingDelegationBalanceEvent),
		party:              party,
		nodeID:             nodeID,
		delegationAmount:   delegationAmount,
		undelegationAmount: undelegationAmount,
		epochSeq:           epochSeq,
	}
}

func (pdb PendingDelegationBalance) Proto() eventspb.PendingDelegationBalanceEvent {
	return eventspb.PendingDelegationBalanceEvent{
		Party:              pdb.party,
		NodeId:             pdb.nodeID,
		DelegationAmount:   pdb.delegationAmount.Uint64(),
		UndelegationAmount: pdb.undelegationAmount.Uint64(),
		EpochSeq:           pdb.epochSeq,
	}
}

func (pdb PendingDelegationBalance) StreamMessage() *eventspb.BusEvent {
	p := pdb.Proto()
	return &eventspb.BusEvent{
		Id:    pdb.eventID(),
		Block: pdb.TraceID(),
		Type:  pdb.et.ToProto(),
		Event: &eventspb.BusEvent_PendingDelegationBalance{
			PendingDelegationBalance: &p,
		},
	}
}

func PendingDelegationBalanceEventFromStream(ctx context.Context, be *eventspb.BusEvent) *PendingDelegationBalance {
	event := be.GetPendingDelegationBalance()
	if event == nil {
		return nil
	}

	return &PendingDelegationBalance{
		Base:               newBaseFromStream(ctx, PendingDelegationBalanceEvent, be),
		party:              event.GetParty(),
		nodeID:             event.GetNodeId(),
		delegationAmount:   num.NewUint(event.GetDelegationAmount()),
		undelegationAmount: num.NewUint(event.GetUndelegationAmount()),
		epochSeq:           event.EpochSeq,
	}
}
