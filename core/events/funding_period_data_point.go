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

type FundingPeriodDataPoint struct {
	*Base
	p *eventspb.FundingPeriodDataPoint
}

func NewFundingPeriodDataPointEvent(ctx context.Context, marketID, mp string, t int64, seq uint64, typ eventspb.FundingPeriodDataPoint_Source) *FundingPeriodDataPoint {
	data := &FundingPeriodDataPoint{
		Base: newBase(ctx, FundingPeriodDataPointEvent),
		p: &eventspb.FundingPeriodDataPoint{
			MarketId:      marketID,
			Price:         mp,
			Timestamp:     t,
			Seq:           seq,
			DataPointType: typ,
		},
	}
	return data
}

func (p *FundingPeriodDataPoint) FundingPeriodDataPoint() *eventspb.FundingPeriodDataPoint {
	return p.p
}

func (p FundingPeriodDataPoint) Proto() eventspb.FundingPeriodDataPoint {
	return *p.p
}

func (p FundingPeriodDataPoint) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_FundingPeriodDataPoint{
		FundingPeriodDataPoint: p.p,
	}
	return busEvent
}

func PeriodicSettlementDataEventFromStream(ctx context.Context, be *eventspb.BusEvent) *FundingPeriodDataPoint {
	return &FundingPeriodDataPoint{
		Base: newBaseFromBusEvent(ctx, EpochUpdate, be),
		p:    be.GetFundingPeriodDataPoint(),
	}
}
