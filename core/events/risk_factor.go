// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
