package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type DelegationBalance struct {
	*Base
	Party    string
	NodeID   string
	Amount   *num.Uint
	EpochSeq string
}

func NewDelegationBalance(ctx context.Context, party, nodeID string, amount *num.Uint, epochSeq string) *DelegationBalance {
	return &DelegationBalance{
		Base:     newBase(ctx, DelegationBalanceEvent),
		Party:    party,
		NodeID:   nodeID,
		Amount:   amount,
		EpochSeq: epochSeq,
	}
}

func (db DelegationBalance) Proto() eventspb.DelegationBalanceEvent {
	return eventspb.DelegationBalanceEvent{
		Party:    db.Party,
		NodeId:   db.NodeID,
		Amount:   db.Amount.String(),
		EpochSeq: db.EpochSeq,
	}
}

func (db DelegationBalance) StreamMessage() *eventspb.BusEvent {
	p := db.Proto()
	return &eventspb.BusEvent{
		Version: eventspb.Version,
		Id:      db.eventID(),
		Block:   db.TraceID(),
		ChainId: db.ChainID(),
		Type:    db.et.ToProto(),
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

	amt, err := num.UintFromString(event.GetAmount(), 10)
	if err {
		return nil
	}

	return &DelegationBalance{
		Base:     newBaseFromStream(ctx, DelegationBalanceEvent, be),
		Party:    event.GetParty(),
		NodeID:   event.GetNodeId(),
		Amount:   amt,
		EpochSeq: event.GetEpochSeq(),
	}
}
