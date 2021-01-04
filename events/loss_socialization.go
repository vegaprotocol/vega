package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

type LossSoc struct {
	*Base
	partyID  string
	marketID string
	amount   int64
	ts       int64
}

func NewLossSocializationEvent(ctx context.Context, partyID, marketID string, amount int64, ts int64) *LossSoc {
	return &LossSoc{
		Base:     newBase(ctx, LossSocializationEvent),
		partyID:  partyID,
		marketID: marketID,
		amount:   amount,
		ts:       ts,
	}
}

func (l LossSoc) IsParty(id string) bool {
	return (l.partyID == id)
}

func (l LossSoc) PartyID() string {
	return l.partyID
}

func (l LossSoc) MarketID() string {
	return l.marketID
}

func (l LossSoc) Amount() int64 {
	return l.amount
}

func (l LossSoc) AmountLost() int64 {
	return l.amount
}

func (l LossSoc) Timestamp() int64 {
	return l.ts
}

func (l LossSoc) Proto() types.LossSocialization {
	return types.LossSocialization{
		MarketID: l.marketID,
		PartyID:  l.partyID,
		Amount:   l.amount,
	}
}

func (l LossSoc) StreamMessage() *types.BusEvent {
	p := l.Proto()
	return &types.BusEvent{
		ID:    l.eventID(),
		Block: l.TraceID(),
		Type:  l.et.ToProto(),
		Event: &types.BusEvent_LossSocialization{
			LossSocialization: &p,
		},
	}
}
