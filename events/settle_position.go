package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
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
	return (s.partyID == id)
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

func (s SettlePos) Proto() types.SettlePosition {
	ts := make([]*types.TradeSettlement, 0, len(s.trades))
	for _, t := range s.trades {
		ts = append(ts, &types.TradeSettlement{
			Size:  t.Size(),
			Price: t.Price(),
		})
	}
	return types.SettlePosition{
		MarketID:         s.marketID,
		PartyID:          s.partyID,
		Price:            s.price,
		TradeSettlements: ts,
	}
}

func (s SettlePos) StreamMessage() *types.BusEvent {
	p := s.Proto()
	return &types.BusEvent{
		ID:   s.eventID(),
		Type: s.et.ToProto(),
		Event: &types.BusEvent_SettlePosition{
			SettlePosition: &p,
		},
	}
}
