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

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type CancelledOrders struct {
	*Base
	pb eventspb.CancelledOrders
}

func NewCancelledOrdersEvent(ctx context.Context, marketID, partyID string, orders ...string) *CancelledOrders {
	return &CancelledOrders{
		Base: newBase(ctx, CancelledOrdersEvent),
		pb: eventspb.CancelledOrders{
			MarketId: marketID,
			PartyId:  partyID,
			OrderIds: orders,
		},
	}
}

func (c CancelledOrders) MarketID() string {
	return c.pb.MarketId
}

func (c CancelledOrders) IsMarket(mID string) bool {
	return c.pb.MarketId == mID
}

func (c CancelledOrders) PartyID() string {
	return c.pb.PartyId
}

func (c CancelledOrders) IsParty(pID string) bool {
	return c.pb.PartyId == pID
}

func (c CancelledOrders) OrderIDs() []string {
	return c.pb.OrderIds
}

func (c CancelledOrders) CompositeCount() uint64 {
	return uint64(len(c.pb.OrderIds))
}

func (c CancelledOrders) Proto() eventspb.CancelledOrders {
	return c.pb
}

func (c CancelledOrders) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(c.Base)
	cpy := c.pb
	busEvent.Event = &eventspb.BusEvent_CancelledOrders{
		CancelledOrders: &cpy,
	}

	return busEvent
}

func (c CancelledOrders) StreamMarketMessage() *eventspb.BusEvent {
	return c.StreamMessage()
}

func CancelledOrdersEventFromStream(ctx context.Context, be *eventspb.BusEvent) *CancelledOrders {
	m := be.GetCancelledOrders()
	return &CancelledOrders{
		Base: newBaseFromBusEvent(ctx, CancelledOrdersEvent, be),
		pb:   *m,
	}
}
