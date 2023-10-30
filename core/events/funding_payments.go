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

type FundingPayments struct {
	*Base
	p *eventspb.FundingPayments
}

func NewFundingPaymentsEvent(ctx context.Context, marketID string, seq uint64, transfers []Transfer) *FundingPayments {
	payments := make([]*eventspb.FundingPayment, 0, len(transfers))
	for _, t := range transfers {
		transfer := t.Transfer()
		pos := true
		if transfer.Type == types.TransferTypePerpFundingLoss {
			pos = false
		}
		amt := num.Numeric{}
		amt.SetInt(num.IntFromUint(transfer.Amount.Amount, pos))
		payments = append(payments, &eventspb.FundingPayment{
			PartyId: t.Party(),
			Amount:  amt.String(),
		})
	}
	interval := &FundingPayments{
		Base: newBase(ctx, FundingPaymentsEvent),
		p: &eventspb.FundingPayments{
			MarketId: marketID,
			Seq:      seq,
			Payments: payments,
		},
	}
	return interval
}

func (p FundingPayments) MarketID() string {
	return p.p.MarketId
}

func (p FundingPayments) IsParty(id string) bool {
	for _, fp := range p.p.Payments {
		if fp.PartyId == id {
			return true
		}
	}
	return false
}

func (p *FundingPayments) FundingPayments() *eventspb.FundingPayments {
	return p.p
}

func (p FundingPayments) Proto() eventspb.FundingPayments {
	return *p.p
}

func (p FundingPayments) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_FundingPayments{
		FundingPayments: p.p,
	}
	return busEvent
}

func FundingPaymentEventFromStream(ctx context.Context, be *eventspb.BusEvent) *FundingPayments {
	return &FundingPayments{
		Base: newBaseFromBusEvent(ctx, FundingPaymentsEvent, be),
		p:    be.GetFundingPayments(),
	}
}
