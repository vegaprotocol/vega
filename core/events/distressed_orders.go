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

// DistressedOrders contains the market and parties that needed to have their orders closed in order
// to maintain their open positions on the market.
type DistressedOrders struct {
	*Base

	pb eventspb.DistressedOrders
}

func NewDistressedOrdersEvent(ctx context.Context, marketID string, parties []string) *DistressedOrders {
	return &DistressedOrders{
		Base: newBase(ctx, DistressedOrdersClosedEvent),
		pb: eventspb.DistressedOrders{
			MarketId: marketID,
			Parties:  parties,
		},
	}
}

func (d DistressedOrders) MarketID() string {
	return d.pb.MarketId
}

func (d DistressedOrders) Parties() []string {
	return d.pb.Parties
}

func (d DistressedOrders) IsMarket(marketID string) bool {
	return d.pb.MarketId == marketID
}

func (d DistressedOrders) IsParty(partyID string) bool {
	for _, p := range d.pb.Parties {
		if p == partyID {
			return true
		}
	}
	return false
}

func (d DistressedOrders) Proto() eventspb.DistressedOrders {
	return d.pb
}

func (d DistressedOrders) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(d.Base)
	cpy := d.pb
	busEvent.Event = &eventspb.BusEvent_DistressedOrders{
		DistressedOrders: &cpy,
	}

	return busEvent
}

func (d DistressedOrders) StreamMarketMessage() *eventspb.BusEvent {
	return d.StreamMessage()
}

func DistressedOrdersEventFromStream(ctx context.Context, be *eventspb.BusEvent) *DistressedOrders {
	m := be.GetDistressedOrders()
	return &DistressedOrders{
		Base: newBaseFromBusEvent(ctx, DistressedOrdersClosedEvent, be),
		pb:   *m,
	}
}
