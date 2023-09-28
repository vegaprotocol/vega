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
