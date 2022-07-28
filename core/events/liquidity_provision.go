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

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/core/types"
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
