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
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	ptypes "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type Order struct {
	*Base
	o *ptypes.Order
}

func NewOrderEvent(ctx context.Context, o *types.Order) *Order {
	order := &Order{
		Base: newBase(ctx, OrderEvent),
		o:    o.IntoProto(),
	}
	// set to original order price
	order.o.Price = num.UintToString(o.OriginalPrice)
	return order
}

func (o Order) IsParty(id string) bool {
	return o.o.PartyId == id
}

func (o Order) PartyID() string {
	return o.o.PartyId
}

func (o Order) MarketID() string {
	return o.o.MarketId
}

func (o *Order) Order() *ptypes.Order {
	return o.o
}

func (o Order) Proto() ptypes.Order {
	return *o.o
}

func (o Order) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(o.Base)
	busEvent.Event = &eventspb.BusEvent_Order{
		Order: o.o,
	}

	// The message is about to be sent out from the core to an external client
	// (which might be the data node)
	o.o.TimeStamps = append(o.o.TimeStamps, &ptypes.SystemTimestamp{Location: 3, TimeStamp: time.Now().UnixNano()})

	return busEvent
}

func OrderEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Order {
	order := &Order{
		Base: newBaseFromBusEvent(ctx, OrderEvent, be),
		o:    be.GetOrder(),
	}
	return order
}
