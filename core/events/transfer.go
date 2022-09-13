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
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

// Transfer ...
type TransferFunds struct {
	*Base
	transferInstruction *eventspb.TransferInstruction
}

func NewOneOffTransferInstructionFundsEvent(
	ctx context.Context,
	t *types.OneOffTransferInstruction,
) *TransferFunds {
	return &TransferFunds{
		Base:                newBase(ctx, TransferInstructionEvent),
		transferInstruction: t.IntoEvent(),
	}
}

func NewRecurringTransferFundsEvent(
	ctx context.Context,
	t *types.RecurringTransferInstruction,
) *TransferFunds {
	return &TransferFunds{
		Base:                newBase(ctx, TransferInstructionEvent),
		transferInstruction: t.IntoEvent(),
	}
}

func (t TransferFunds) PartyID() string {
	return t.transferInstruction.From
}

func (t TransferFunds) TransferFunds() eventspb.TransferInstruction {
	return t.Proto()
}

func (t TransferFunds) Proto() eventspb.TransferInstruction {
	return *t.transferInstruction
}

func (t TransferFunds) StreamMessage() *eventspb.BusEvent {
	p := t.Proto()

	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_TransferInstruction{
		TransferInstruction: &p,
	}

	return busEvent
}

func TransferFundsEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TransferFunds {
	event := be.GetTransferInstruction()
	if event == nil {
		return nil
	}

	return &TransferFunds{
		Base:                newBaseFromBusEvent(ctx, TransferInstructionEvent, be),
		transferInstruction: event,
	}
}
