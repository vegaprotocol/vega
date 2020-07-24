package events

import "context"

type LossSoc struct {
	*Base
	partyID  string
	marketID string
	amount   int64
}

func NewLossSocializationEvent(ctx context.Context, partyID, marketID string, amount int64) *LossSoc {
	return &LossSoc{
		Base:     newBase(ctx, LossSocializationEvent),
		partyID:  partyID,
		marketID: marketID,
		amount:   amount,
	}
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
