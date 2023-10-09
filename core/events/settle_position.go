// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package events

import (
	"context"

	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type SettlePos struct {
	*Base
	partyID        string
	marketID       string
	positionFactor num.Decimal
	price          *num.Uint
	trades         []TradeSettlement
	ts             int64
}

func NewSettlePositionEvent(ctx context.Context, partyID, marketID string, price *num.Uint, trades []TradeSettlement, ts int64, positionFactor num.Decimal) *SettlePos {
	return &SettlePos{
		Base:           newBase(ctx, SettlePositionEvent),
		partyID:        partyID,
		marketID:       marketID,
		price:          price.Clone(),
		trades:         trades,
		ts:             ts,
		positionFactor: positionFactor,
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

func (s SettlePos) PositionFactor() num.Decimal {
	return s.positionFactor
}

func (s SettlePos) Proto() eventspb.SettlePosition {
	ts := make([]*eventspb.TradeSettlement, 0, len(s.trades))
	for _, t := range s.trades {
		ts = append(ts, &eventspb.TradeSettlement{
			Size:        t.Size(),
			Price:       t.Price().String(),
			MarketPrice: t.MarketPrice().String(),
		})
	}
	return eventspb.SettlePosition{
		MarketId:         s.marketID,
		PartyId:          s.partyID,
		Price:            s.price.String(),
		PositionFactor:   s.positionFactor.String(),
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
	SettlementSize        int64
	SettlementPrice       *num.Uint
	SettlementMarketPrice *num.Uint
}

func (s settlement) Size() int64 {
	return s.SettlementSize
}

func (s settlement) Price() *num.Uint {
	return s.SettlementPrice
}

func (s settlement) MarketPrice() *num.Uint {
	return s.SettlementMarketPrice
}

func SettlePositionEventFromStream(ctx context.Context, be *eventspb.BusEvent) *SettlePos {
	sp := be.GetSettlePosition()
	settlements := make([]TradeSettlement, 0, len(sp.TradeSettlements))
	for _, ts := range sp.TradeSettlements {
		price, _ := num.UintFromString(ts.Price, 10)
		marketPrice, _ := num.UintFromString(ts.MarketPrice, 10)
		settlements = append(settlements, settlement{
			SettlementSize:        ts.Size,
			SettlementPrice:       price,
			SettlementMarketPrice: marketPrice,
		})
	}
	spPrice, _ := num.UintFromString(sp.Price, 10)
	positionFactor := num.MustDecimalFromString(sp.PositionFactor)

	return &SettlePos{
		Base:           newBaseFromBusEvent(ctx, SettlePositionEvent, be),
		partyID:        sp.PartyId,
		marketID:       sp.MarketId,
		price:          spPrice,
		trades:         settlements,
		positionFactor: positionFactor,
	}
}
