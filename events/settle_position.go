package events

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

type SettlePos struct {
	*Base
	partyID  string
	marketID string
	price    uint64
	trades   []TradeSettlement
	ts       int64
}

func NewSettlePositionEvent(ctx context.Context, partyID, marketID string, price uint64, trades []TradeSettlement, ts int64) *SettlePos {
	return &SettlePos{
		Base:     newBase(ctx, SettlePositionEvent),
		partyID:  partyID,
		marketID: marketID,
		price:    price,
		trades:   trades,
		ts:       ts,
	}
}

func (s SettlePos) MarketID() string {
	return s.marketID
}

func (s SettlePos) IsParty(id string) bool {
	return s.partyID == id
}

func (s SettlePos) PartyID() string {
	return s.partyID
}

func (s SettlePos) Price() uint64 {
	return s.price
}

func (s SettlePos) Trades() []TradeSettlement {
	return s.trades
}

func (s SettlePos) Timestamp() int64 {
	return s.ts
}

func (s SettlePos) Proto() eventspb.SettlePosition {
	ts := make([]*eventspb.TradeSettlement, 0, len(s.trades))
	for _, t := range s.trades {
		ts = append(ts, &eventspb.TradeSettlement{
			Size:  t.Size(),
			Price: t.Price(),
		})
	}
	return eventspb.SettlePosition{
		MarketId:         s.marketID,
		PartyId:          s.partyID,
		Price:            s.price,
		TradeSettlements: ts,
	}
}

func (s SettlePos) StreamMessage() *eventspb.BusEvent {
	p := s.Proto()
	return &eventspb.BusEvent{
		Id:    s.eventID(),
		Block: s.TraceID(),
		Type:  s.et.ToProto(),
		Event: &eventspb.BusEvent_SettlePosition{
			SettlePosition: &p,
		},
	}
}
