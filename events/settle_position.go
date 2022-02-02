package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type SettlePos struct {
	*Base
	partyID  string
	marketID string
	price    *num.Uint
	trades   []TradeSettlement
	ts       int64
}

func NewSettlePositionEvent(ctx context.Context, partyID, marketID string, price *num.Uint, trades []TradeSettlement, ts int64) *SettlePos {
	return &SettlePos{
		Base:     newBase(ctx, SettlePositionEvent),
		partyID:  partyID,
		marketID: marketID,
		price:    price.Clone(),
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

func (s SettlePos) Price() *num.Uint {
	return s.price.Clone()
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
			Price: t.Price().String(),
		})
	}
	return eventspb.SettlePosition{
		MarketId:         s.marketID,
		PartyId:          s.partyID,
		Price:            s.price.String(),
		TradeSettlements: ts,
	}
}

func (s SettlePos) StreamMessage() *eventspb.BusEvent {
	p := s.Proto()

	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_SettlePosition{
		SettlePosition: &p,
	}

	return busEvent
}

type settlement struct {
	SettlementSize  int64
	SettlementPrice *num.Uint
}

func (s settlement) Size() int64 {
	return s.SettlementSize
}

func (s settlement) Price() *num.Uint {
	return s.SettlementPrice
}

func SettlePositionEventFromStream(ctx context.Context, be *eventspb.BusEvent) *SettlePos {
	sp := be.GetSettlePosition()
	settlements := make([]TradeSettlement, 0, len(sp.TradeSettlements))
	for _, ts := range sp.TradeSettlements {
		price, _ := num.UintFromString(ts.Price, 10)
		settlements = append(settlements, settlement{
			SettlementSize:  ts.Size,
			SettlementPrice: price,
		})
	}
	spPrice, _ := num.UintFromString(sp.Price, 10)
	return &SettlePos{
		Base:     newBaseFromBusEvent(ctx, SettlePositionEvent, be),
		partyID:  sp.PartyId,
		marketID: sp.MarketId,
		price:    spPrice,
		trades:   settlements,
	}
}
