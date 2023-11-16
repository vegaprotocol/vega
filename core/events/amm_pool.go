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
	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type AMMPool struct {
	*Base
	pool *eventspb.AMMPool
}

func NewAMMPoolEvent(
	ctx context.Context,
	party, market, subAccount, poolID string,
	commitment *num.Uint,
	p *types.ConcentratedLiquidityParameters,
	status types.AMMPoolStatus,
	statusReason types.AMMPoolStatusReason,
) *AMMPool {
	order := &AMMPool{
		Base: newBase(ctx, AMMPoolEvent),
		pool: &eventspb.AMMPool{
			PartyId:      party,
			MarketId:     market,
			PoolId:       poolID,
			SubAccount:   subAccount,
			Commitment:   commitment.String(),
			Parameters:   p.ToProtoEvent(),
			Status:       status,
			StatusReason: statusReason,
		},
	}
	// set to original order price
	return order
}

func (p AMMPool) IsParty(id string) bool {
	return p.pool.PartyId == id
}

func (p AMMPool) PartyID() string {
	return p.pool.PartyId
}

func (p AMMPool) MarketID() string {
	return p.pool.MarketId
}

func (p *AMMPool) AMMPool() *eventspb.AMMPool {
	return p.pool
}

func (p AMMPool) Proto() eventspb.AMMPool {
	return *p.pool
}

func (p AMMPool) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_AmmPool{
		AmmPool: p.pool,
	}

	return busEvent
}

func AMMPoolEventFromStream(ctx context.Context, be *eventspb.BusEvent) *AMMPool {
	pool := &AMMPool{
		Base: newBaseFromBusEvent(ctx, AMMPoolEvent, be),
		pool: be.GetAmmPool(),
	}
	return pool
}
