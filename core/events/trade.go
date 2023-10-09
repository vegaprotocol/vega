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
