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

type RiskFactor struct {
	*Base
	r proto.RiskFactor
}

func NewRiskFactorEvent(ctx context.Context, r types.RiskFactor) *RiskFactor {
	return &RiskFactor{
		Base: newBase(ctx, RiskFactorEvent),
		r:    *r.IntoProto(),
	}
}

func (r RiskFactor) MarketID() string {
	return r.r.Market
}

func (r *RiskFactor) RiskFactor() proto.RiskFactor {
	return r.r
}

func (r RiskFactor) Proto() proto.RiskFactor {
	return r.r
}

func (r RiskFactor) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(r.Base)
	busEvent.Event = &eventspb.BusEvent_RiskFactor{
		RiskFactor: &r.r,
	}

	return busEvent
}

func RiskFactorEventFromStream(ctx context.Context, be *eventspb.BusEvent) *RiskFactor {
	return &RiskFactor{
		Base: newBaseFromBusEvent(ctx, RiskFactorEvent, be),
		r:    *be.GetRiskFactor(),
	}
}
