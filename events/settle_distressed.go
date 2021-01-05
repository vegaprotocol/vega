package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

type SettleDistressed struct {
	*Base
	partyID  string
	marketID string
	margin   uint64
	price    uint64
	ts       int64
}

func NewSettleDistressed(ctx context.Context, partyID, marketID string, price, margin uint64, ts int64) *SettleDistressed {
	return &SettleDistressed{
		Base:     newBase(ctx, SettleDistressedEvent),
		partyID:  partyID,
		marketID: marketID,
		margin:   margin,
		price:    price,
		ts:       ts,
	}
}

func (s SettleDistressed) IsParty(id string) bool {
	return (s.partyID == id)
}

func (s SettleDistressed) PartyID() string {
	return s.partyID
}

func (s SettleDistressed) MarketID() string {
	return s.marketID
}

func (s SettleDistressed) Margin() uint64 {
	return s.margin
}

func (s SettleDistressed) Price() uint64 {
	return s.price
}

func (s SettleDistressed) Timestamp() int64 {
	return s.ts
}

func (s SettleDistressed) Proto() types.SettleDistressed {
	return types.SettleDistressed{
		MarketID: s.marketID,
		PartyID:  s.partyID,
		Margin:   s.margin,
		Price:    s.price,
	}
}

func (s SettleDistressed) StreamMessage() *types.BusEvent {
	p := s.Proto()
	return &types.BusEvent{
		ID:    s.eventID(),
		Block: s.TraceID(),
		Type:  s.et.ToProto(),
		Event: &types.BusEvent_SettleDistressed{
			SettleDistressed: &p,
		},
	}
}
