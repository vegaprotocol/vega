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
