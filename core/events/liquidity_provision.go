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
	proto "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type LiquidityProvision struct {
	*Base
	p *proto.LiquidityProvision
}

func NewLiquidityProvisionEvent(ctx context.Context, p *types.LiquidityProvision) *LiquidityProvision {
	order := &LiquidityProvision{
		Base: newBase(ctx, LiquidityProvisionEvent),
		p:    p.IntoProto(),
	}
	return order
}

func (p LiquidityProvision) IsParty(id string) bool {
	return p.p.PartyId == id
}

func (p LiquidityProvision) PartyID() string {
	return p.p.PartyId
}

func (p LiquidityProvision) MarketID() string {
	return p.p.MarketId
}

func (p LiquidityProvision) LiquidityProvision() *proto.LiquidityProvision {
	return p.p
}

func (p LiquidityProvision) Proto() *proto.LiquidityProvision {
	return p.p
}

func (p LiquidityProvision) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_LiquidityProvision{
		LiquidityProvision: p.p,
	}

	return busEvent
}

func LiquidityProvisionEventFromStream(ctx context.Context, be *eventspb.BusEvent) *LiquidityProvision {
	order := &LiquidityProvision{
		Base: newBaseFromBusEvent(ctx, LiquidityProvisionEvent, be),
		p:    be.GetLiquidityProvision(),
	}
	return order
}
