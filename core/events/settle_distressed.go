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
		Margin:   s.margin.String(),
		Price:    s.price.String(),
	}
}

func (s SettleDistressed) StreamMessage() *eventspb.BusEvent {
	p := s.Proto()

	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_SettleDistressed{
		SettleDistressed: &p,
	}

	return busEvent
}

func SettleDistressedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *SettleDistressed {
	sd := be.GetSettleDistressed()
	sdMargin, marginOverflow := num.UintFromString(sd.Margin, 10)
	sdPrice, priceOverflow := num.UintFromString(sd.Price, 10)

	if marginOverflow || priceOverflow {
		return nil
	}

	return &SettleDistressed{
		Base:     newBaseFromBusEvent(ctx, SettleDistressedEvent, be),
		partyID:  sd.PartyId,
		marketID: sd.MarketId,
		margin:   sdMargin,
		price:    sdPrice,
	}
}
