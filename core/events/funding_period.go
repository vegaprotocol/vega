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
