package events

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type SettleDistressed struct {
	*Base
	partyID  string
	marketID string
	margin   *num.Uint
	price    *num.Uint
	ts       int64
}

func NewSettleDistressed(ctx context.Context, partyID, marketID string, price, margin *num.Uint, ts int64) *SettleDistressed {
	return &SettleDistressed{
		Base:     newBase(ctx, SettleDistressedEvent),
		partyID:  partyID,
		marketID: marketID,
		margin:   margin.Clone(),
		price:    price.Clone(),
		ts:       ts,
	}
}

func (s SettleDistressed) IsParty(id string) bool {
	return s.partyID == id
}

func (s SettleDistressed) PartyID() string {
	return s.partyID
}

func (s SettleDistressed) MarketID() string {
	return s.marketID
}

func (s SettleDistressed) Margin() *num.Uint {
	return s.margin.Clone()
}

func (s SettleDistressed) Price() *num.Uint {
	return s.price.Clone()
}

func (s SettleDistressed) Timestamp() int64 {
	return s.ts
}

func (s SettleDistressed) Proto() eventspb.SettleDistressed {
	return eventspb.SettleDistressed{
		MarketId: s.marketID,
		PartyId:  s.partyID,
		Margin:   s.margin.Uint64(),
		Price:    s.price.Uint64(),
	}
}

func (s SettleDistressed) StreamMessage() *eventspb.BusEvent {
	p := s.Proto()
	return &eventspb.BusEvent{
		Id:    s.eventID(),
		Block: s.TraceID(),
		Type:  s.et.ToProto(),
		Event: &eventspb.BusEvent_SettleDistressed{
			SettleDistressed: &p,
		},
	}
}
