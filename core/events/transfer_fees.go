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
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

// TransferFees ...
type TransferFees struct {
	*Base
	pb *eventspb.TransferFees
}

func NewTransferFeesEvent(ctx context.Context, transferID string, amount *num.Uint, discount *num.Uint, epoch uint64) *TransferFees {
	return &TransferFees{
		Base: newBase(ctx, TransferFeesEvent),
		pb: &eventspb.TransferFees{
			TransferId:      transferID,
			Amount:          amount.String(),
			DiscountApplied: discount.String(),
			Epoch:           epoch,
		},
	}
}

func (t TransferFees) Proto() eventspb.TransferFees {
	return *t.pb
}

func (t TransferFees) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_TransferFees{
		TransferFees: t.pb,
	}

	return busEvent
}

func (t TransferFees) TransferFees() eventspb.TransferFees {
	return t.Proto()
}

func TransferFeesEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TransferFees {
	event := be.GetTransferFees()
	if event == nil {
		return nil
	}

	return &TransferFees{
		Base: newBaseFromBusEvent(ctx, TransferFeesEvent, be),
		pb:   event,
	}
}

// TransferFeesDiscountUpdated ...
type TransferFeesDiscountUpdated struct {
	*Base
	pb *eventspb.TransferFeesDiscount
}

func NewTransferFeesDiscountUpdated(ctx context.Context, party, asset string, amount *num.Uint, epoch uint64) *TransferFeesDiscountUpdated {
	return &TransferFeesDiscountUpdated{
		Base: newBase(ctx, TransferFeesDiscountUpdatedEvent),
		pb: &eventspb.TransferFeesDiscount{
			Party:  party,
			Asset:  asset,
			Amount: amount.String(),
			Epoch:  epoch,
		},
	}
}

func (t TransferFeesDiscountUpdated) Proto() eventspb.TransferFeesDiscount {
	return *t.pb
}

func (t TransferFeesDiscountUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_TransferFeesDiscount{
		TransferFeesDiscount: t.pb,
	}

	fmt.Printf("-------- stream message jare: %+v \n", busEvent)

	return busEvent
}

func (t TransferFeesDiscountUpdated) TransferFeesDiscount() eventspb.TransferFeesDiscount {
	return t.Proto()
}

func TransferFeesDiscountUpdatedFromStream(ctx context.Context, be *eventspb.BusEvent) *TransferFeesDiscountUpdated {
	event := be.GetTransferFeesDiscount()
	if event == nil {
		return nil
	}

	return &TransferFeesDiscountUpdated{
		Base: newBaseFromBusEvent(ctx, TransferFeesDiscountUpdatedEvent, be),
		pb:   event,
	}
}
