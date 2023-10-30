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

type PositionState struct {
	*Base
	partyID        string
	marketID       string
	size           int64
	potentialBuys  int64
	potentialSells int64
	vwBuyPrice     *num.Uint
	vwSellPrice    *num.Uint
}

func NewPositionStateEvent(ctx context.Context, mp MarketPosition, marketID string) *PositionState {
	return &PositionState{
		Base:           newBase(ctx, PositionStateEvent),
		partyID:        mp.Party(),
		marketID:       marketID,
		size:           mp.Size(),
		potentialBuys:  mp.Buy(),
		potentialSells: mp.Sell(),
		vwBuyPrice:     mp.VWBuy(),
		vwSellPrice:    mp.VWSell(),
	}
}

func (s PositionState) MarketID() string {
	return s.marketID
}

func (s PositionState) IsParty(id string) bool {
	return s.partyID == id
}

func (s PositionState) PartyID() string {
	return s.partyID
}

func (s PositionState) Size() int64 {
	return s.size
}

func (s PositionState) PotentialBuys() int64 {
	return s.potentialBuys
}

func (s PositionState) PotentialSells() int64 {
	return s.potentialSells
}

func (s PositionState) VWBuyPrice() *num.Uint {
	return s.vwBuyPrice
}

func (s PositionState) VWSellPrice() *num.Uint {
	return s.vwSellPrice
}

func (s PositionState) Proto() eventspb.PositionStateEvent {
	return eventspb.PositionStateEvent{
		MarketId:       s.marketID,
		PartyId:        s.partyID,
		Size:           s.size,
		PotentialBuys:  s.potentialBuys,
		PotentialSells: s.potentialSells,
		VwBuyPrice:     s.vwBuyPrice.String(),
		VwSellPrice:    s.vwSellPrice.String(),
	}
}

func (s PositionState) StreamMessage() *eventspb.BusEvent {
	p := s.Proto()

	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_PositionStateEvent{
		PositionStateEvent: &p,
	}

	return busEvent
}

func PositionStateEventFromStream(ctx context.Context, be *eventspb.BusEvent) *PositionState {
	pse := be.GetPositionStateEvent()

	vwBuy, overflow := num.UintFromString(pse.VwBuyPrice, 10)
	if overflow {
		return nil
	}

	vwSell, overflow := num.UintFromString(pse.VwSellPrice, 10)
	if overflow {
		return nil
	}

	return &PositionState{
		Base:           newBaseFromBusEvent(ctx, PositionStateEvent, be),
		partyID:        pse.PartyId,
		marketID:       pse.MarketId,
		size:           pse.Size,
		potentialBuys:  pse.PotentialBuys,
		potentialSells: pse.PotentialSells,
		vwBuyPrice:     vwBuy,
		vwSellPrice:    vwSell,
	}
}
