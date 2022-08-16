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

	"code.vegaprotocol.io/vega/core/types"
	ptypes "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type Trade struct {
	*Base
	t ptypes.Trade
}

func NewTradeEvent(ctx context.Context, t types.Trade) *Trade {
	p := t.IntoProto()
	p.Price = t.MarketPrice.String()
	return &Trade{
		Base: newBase(ctx, TradeEvent),
		t:    *p,
	}
}

func (t Trade) MarketID() string {
	return t.t.MarketId
}

func (t Trade) IsParty(id string) bool {
	return t.t.Buyer == id || t.t.Seller == id
}

func (t *Trade) Trade() ptypes.Trade {
	return t.t
}

func (t Trade) Proto() ptypes.Trade {
	return t.t
}

func (t Trade) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_Trade{
		Trade: &t.t,
	}

	return busEvent
}

func TradeEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Trade {
	return &Trade{
		Base: newBaseFromBusEvent(ctx, TradeEvent, be),
		t:    *be.GetTrade(),
	}
}
