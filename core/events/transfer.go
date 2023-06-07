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

// Transfer ...
type TransferFunds struct {
	*Base
	transfer *eventspb.Transfer
}

func NewGovTransferFundsEvent(
	ctx context.Context,
	t *types.GovernanceTransfer,
	amount *num.Uint,
) *TransferFunds {
	return &TransferFunds{
		Base:     newBase(ctx, TransferEvent),
		transfer: t.IntoEvent(amount, nil),
	}
}

func NewGovTransferFundsEventWithReason(
	ctx context.Context,
	t *types.GovernanceTransfer,
	amount *num.Uint,
	reason string,
) *TransferFunds {
	return &TransferFunds{
		Base:     newBase(ctx, TransferEvent),
		transfer: t.IntoEvent(amount, &reason),
	}
}

func NewOneOffTransferFundsEvent(
	ctx context.Context,
	t *types.OneOffTransfer,
) *TransferFunds {
	return &TransferFunds{
		Base:     newBase(ctx, TransferEvent),
		transfer: t.IntoEvent(nil),
	}
}

func NewOneOffTransferFundsEventWithReason(
	ctx context.Context,
	t *types.OneOffTransfer,
	reason string,
) *TransferFunds {
	return &TransferFunds{
		Base:     newBase(ctx, TransferEvent),
		transfer: t.IntoEvent(&reason),
	}
}

func NewRecurringTransferFundsEvent(
	ctx context.Context,
	t *types.RecurringTransfer,
) *TransferFunds {
	return &TransferFunds{
		Base:     newBase(ctx, TransferEvent),
		transfer: t.IntoEvent(nil),
	}
}

func NewRecurringTransferFundsEventWithReason(
	ctx context.Context,
	t *types.RecurringTransfer,
	reason string,
) *TransferFunds {
	return &TransferFunds{
		Base:     newBase(ctx, TransferEvent),
		transfer: t.IntoEvent(&reason),
	}
}

func (t TransferFunds) PartyID() string {
	return t.transfer.From
}

func (t TransferFunds) TransferFunds() eventspb.Transfer {
	return t.Proto()
}

func (t TransferFunds) Proto() eventspb.Transfer {
	return *t.transfer
}

func (t TransferFunds) StreamMessage() *eventspb.BusEvent {
	p := t.Proto()

	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_Transfer{
		Transfer: &p,
	}

	return busEvent
}

func TransferFundsEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TransferFunds {
	event := be.GetTransfer()
	if event == nil {
		return nil
	}

	return &TransferFunds{
		Base:     newBaseFromBusEvent(ctx, TransferEvent, be),
		transfer: event,
	}
}
