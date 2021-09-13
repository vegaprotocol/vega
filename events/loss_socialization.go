package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type LossSoc struct {
	*Base
	partyID  string
	marketID string
	amount   *num.Uint
	neg      bool
	ts       int64
}

func NewLossSocializationEvent(ctx context.Context, partyID, marketID string, amount *num.Uint, neg bool, ts int64) *LossSoc {
	return &LossSoc{
		Base:     newBase(ctx, LossSocializationEvent),
		partyID:  partyID,
		marketID: marketID,
		amount:   amount,
		neg:      neg,
		ts:       ts,
	}
}

func (l LossSoc) IsParty(id string) bool {
	return l.partyID == id
}

func (l LossSoc) PartyID() string {
	return l.partyID
}

func (l LossSoc) MarketID() string {
	return l.marketID
}

func (l LossSoc) Negative() bool {
	return l.neg
}

func (l LossSoc) Amount() *num.Uint {
	return l.amount.Clone()
}

func (l LossSoc) Timestamp() int64 {
	return l.ts
}

func (l LossSoc) Proto() eventspb.LossSocialization {
	amt := l.amount.String()
	if l.neg {
		amt = "-" + amt
	}
	return eventspb.LossSocialization{
		MarketId: l.marketID,
		PartyId:  l.partyID,
		Amount:   amt,
	}
}

func (l LossSoc) StreamMessage() *eventspb.BusEvent {
	p := l.Proto()
	return &eventspb.BusEvent{
		Version: eventspb.Version,
		Id:      l.eventID(),
		Block:   l.TraceID(),
		Type:    l.et.ToProto(),
		Event: &eventspb.BusEvent_LossSocialization{
			LossSocialization: &p,
		},
	}
}

func LossSocializationEventFromStream(ctx context.Context, be *eventspb.BusEvent) *LossSoc {
	lse := &LossSoc{
		Base:     newBaseFromStream(ctx, LossSocializationEvent, be),
		partyID:  be.GetLossSocialization().PartyId,
		marketID: be.GetLossSocialization().MarketId,
	}

	amt := be.GetLossSocialization().Amount
	if len(amt) > 0 && amt[0] == '-' {
		lse.neg = true
		amt = amt[1:]
		lse.amount, _ = num.UintFromString(amt, 10)
		return lse
	}

	lse.amount, _ = num.UintFromString(amt, 10)
	return lse
}
