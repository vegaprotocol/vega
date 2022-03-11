package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
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

	vwBuy, ok := num.UintFromString(pse.VwBuyPrice, 10)
	if !ok {
		return nil
	}

	vwSell, ok := num.UintFromString(pse.VwSellPrice, 10)
	if !ok {
		return nil
	}

	return &PositionState{
		Base:           newBaseFromBusEvent(ctx, SettlePositionEvent, be),
		partyID:        pse.PartyId,
		marketID:       pse.MarketId,
		size:           pse.Size,
		potentialBuys:  pse.PotentialBuys,
		potentialSells: pse.PotentialSells,
		vwBuyPrice:     vwBuy,
		vwSellPrice:    vwSell,
	}
}
