// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/core/types/num"
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
