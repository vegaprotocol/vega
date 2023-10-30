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

	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type FundingPeriodDataPoint struct {
	*Base
	p *eventspb.FundingPeriodDataPoint
}

func NewFundingPeriodDataPointEvent(ctx context.Context, marketID, mp string, t int64, seq uint64, typ eventspb.FundingPeriodDataPoint_Source, twap *num.Uint) *FundingPeriodDataPoint {
	data := &FundingPeriodDataPoint{
		Base: newBase(ctx, FundingPeriodDataPointEvent),
		p: &eventspb.FundingPeriodDataPoint{
			MarketId:      marketID,
			Price:         mp,
			Timestamp:     t,
			Seq:           seq,
			DataPointType: typ,
			Twap:          twap.String(),
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

func FundingPeriodDataPointEventFromStream(ctx context.Context, be *eventspb.BusEvent) *FundingPeriodDataPoint {
	return &FundingPeriodDataPoint{
		Base: newBaseFromBusEvent(ctx, FundingPeriodDataPointEvent, be),
		p:    be.GetFundingPeriodDataPoint(),
	}
}
