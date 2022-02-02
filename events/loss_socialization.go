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
	amount   *num.Int
	ts       int64
}

func NewLossSocializationEvent(ctx context.Context, partyID, marketID string, amount *num.Uint, neg bool, ts int64) *LossSoc {
	signedAmount := num.NewIntFromUint(amount)
	if neg {
		signedAmount.FlipSign()
	}
	return &LossSoc{
		Base:     newBase(ctx, LossSocializationEvent),
		partyID:  partyID,
		marketID: marketID,
		amount:   signedAmount,
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
	return l.amount.IsNegative()
}

func (l LossSoc) Amount() *num.Int {
	return l.amount.Clone()
}

func (l LossSoc) Timestamp() int64 {
	return l.ts
}

func (l LossSoc) Proto() eventspb.LossSocialization {
	return eventspb.LossSocialization{
		MarketId: l.marketID,
		PartyId:  l.partyID,
		Amount:   l.amount.String(),
	}
}

func (l LossSoc) StreamMessage() *eventspb.BusEvent {
	p := l.Proto()

	busEvent := newBusEventFromBase(l.Base)
	busEvent.Event = &eventspb.BusEvent_LossSocialization{
		LossSocialization: &p,
	}

	return busEvent
}

func LossSocializationEventFromStream(ctx context.Context, be *eventspb.BusEvent) *LossSoc {
	lse := &LossSoc{
		Base:     newBaseFromBusEvent(ctx, LossSocializationEvent, be),
		partyID:  be.GetLossSocialization().PartyId,
		marketID: be.GetLossSocialization().MarketId,
	}

	lse.amount, _ = num.IntFromString(be.GetLossSocialization().Amount, 10)
	return lse
}
