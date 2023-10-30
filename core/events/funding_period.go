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

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type FundingPeriod struct {
	*Base
	p *eventspb.FundingPeriod
}

func NewFundingPeriodEvent(ctx context.Context, marketID string, seq uint64, start int64, end *int64, fundingPayment, fundingRate, iTWAP, eTWAP *string) *FundingPeriod {
	interval := &FundingPeriod{
		Base: newBase(ctx, FundingPeriodEvent),
		p: &eventspb.FundingPeriod{
			MarketId:       marketID,
			Start:          start,
			End:            end,
			FundingPayment: fundingPayment,
			FundingRate:    fundingRate,
			Seq:            seq,
			InternalTwap:   iTWAP,
			ExternalTwap:   eTWAP,
		},
	}
	return interval
}

func (p *FundingPeriod) FundingPeriod() *eventspb.FundingPeriod {
	return p.p
}

func (p FundingPeriod) Proto() eventspb.FundingPeriod {
	return *p.p
}

func (p FundingPeriod) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_FundingPeriod{
		FundingPeriod: p.p,
	}
	return busEvent
}

func FundingPeriodEventFromStream(ctx context.Context, be *eventspb.BusEvent) *FundingPeriod {
	return &FundingPeriod{
		Base: newBaseFromBusEvent(ctx, FundingPeriodEvent, be),
		p:    be.GetFundingPeriod(),
	}
}
