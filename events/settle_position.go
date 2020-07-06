package events

import "context"

type SettlePos struct {
	*Base
	partyID  string
	marketID string
	price    uint64
	trades   []TradeSettlement
}

func NewSettlePositionEvent(ctx context.Context, partyID, marketID string, price uint64, trades []TradeSettlement) *SettlePos {
	return &SettlePos{
		Base:     newBase(ctx, SettlePositionEvent),
		partyID:  partyID,
		marketID: marketID,
		price:    price,
		trades:   trades,
	}
}

func (s SettlePos) MarketID() string {
	return s.marketID
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
