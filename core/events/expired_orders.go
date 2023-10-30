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

// ExpiredOrders contains the market and parties that needed to have their orders closed in order
// to maintain their open positions on the market.
type ExpiredOrders struct {
	*Base

	pb eventspb.ExpiredOrders
}

func NewExpiredOrdersEvent(ctx context.Context, marketID string, orders []string) *ExpiredOrders {
	return &ExpiredOrders{
		Base: newBase(ctx, ExpiredOrdersEvent),
		pb: eventspb.ExpiredOrders{
			MarketId: marketID,
			OrderIds: orders,
		},
	}
}

func (d ExpiredOrders) MarketID() string {
	return d.pb.MarketId
}

func (d ExpiredOrders) OrderIDs() []string {
	return d.pb.OrderIds
}

func (d ExpiredOrders) CompositeCount() uint64 {
	return uint64(len(d.pb.OrderIds))
}

func (d ExpiredOrders) IsMarket(marketID string) bool {
	return d.pb.MarketId == marketID
}

func (d ExpiredOrders) Proto() eventspb.ExpiredOrders {
	return d.pb
}

func (d ExpiredOrders) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(d.Base)
	cpy := d.pb
	busEvent.Event = &eventspb.BusEvent_ExpiredOrders{
		ExpiredOrders: &cpy,
	}

	return busEvent
}

func (d ExpiredOrders) StreamMarketMessage() *eventspb.BusEvent {
	return d.StreamMessage()
}

func ExpiredOrdersEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ExpiredOrders {
	m := be.GetExpiredOrders()
	return &ExpiredOrders{
		Base: newBaseFromBusEvent(ctx, ExpiredOrdersEvent, be),
		pb:   *m,
	}
}
