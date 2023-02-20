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
