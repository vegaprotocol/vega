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
	pool *eventspb.AMM
}

type AMMCurve struct {
	VirtualLiquidity    num.Decimal
	TheoreticalPosition num.Decimal
}

func (a *AMMCurve) ToProtoEvent() *eventspb.AMM_Curve {
	if a == nil {
		return nil
	}

	return &eventspb.AMM_Curve{
		VirtualLiquidity:    a.VirtualLiquidity.String(),
		TheoreticalPosition: a.TheoreticalPosition.String(),
	}
}

func NewAMMPoolEvent(
	ctx context.Context,
	party, market, ammParty, poolID string,
	commitment *num.Uint,
	p *types.ConcentratedLiquidityParameters,
	status types.AMMPoolStatus,
	statusReason types.AMMStatusReason,
	fees num.Decimal,
	lowerCurve *AMMCurve,
	upperCurve *AMMCurve,
	minimumPriceChangeTrigger num.Decimal,
) *AMMPool {
	return &AMMPool{
		Base: newBase(ctx, AMMPoolEvent),
		pool: &eventspb.AMM{
			Id:                        poolID,
			PartyId:                   party,
			MarketId:                  market,
			AmmPartyId:                ammParty,
			Commitment:                commitment.String(),
			Parameters:                p.ToProtoEvent(),
			Status:                    status,
			StatusReason:              statusReason,
			ProposedFee:               fees.String(),
			LowerCurve:                lowerCurve.ToProtoEvent(),
			UpperCurve:                upperCurve.ToProtoEvent(),
			MinimumPriceChangeTrigger: minimumPriceChangeTrigger.String(),
		},
	}
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

func (p *AMMPool) AMMPool() *eventspb.AMM {
	return p.pool
}

func (p AMMPool) Proto() eventspb.AMM {
	return *p.pool
}

func (p AMMPool) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_Amm{
		Amm: p.pool,
	}

	return busEvent
}

func AMMPoolEventFromStream(ctx context.Context, be *eventspb.BusEvent) *AMMPool {
	pool := &AMMPool{
		Base: newBaseFromBusEvent(ctx, AMMPoolEvent, be),
		pool: be.GetAmm(),
	}
	return pool
}
