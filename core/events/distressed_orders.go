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
