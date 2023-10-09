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

package banking

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

var (
	ErrRecurringTransferDoesNotExists             = errors.New("recurring transfer does not exists")
	ErrCannotCancelOtherPartiesRecurringTransfers = errors.New("cannot cancel other parties recurring transfers")
)

func (e *Engine) CancelTransferFunds(
	ctx context.Context,
	cancel *types.CancelTransferFunds,
) error {
	// validation is simple, does the transfer
	// exists
	transfer, ok := e.recurringTransfersMap[cancel.TransferID]
	if !ok {
		return ErrRecurringTransferDoesNotExists
	}

	// Is the From party of the transfer
	// the party which submitted the transaction?
	if transfer.From != cancel.Party {
		return ErrCannotCancelOtherPartiesRecurringTransfers
	}

	// all good, let's delete
	e.deleteTransfer(cancel.TransferID)

	// send an event because we are nice with the data-node
	transfer.Status = types.TransferStatusCancelled
	e.broker.Send(events.NewRecurringTransferFundsEventWithReason(ctx, transfer, "transfer cancelled"))

	return nil
}

func (e *Engine) CancelGovTransfer(ctx context.Context, ID string) error {
	gTransfer, ok := e.recurringGovernanceTransfersMap[ID]
	if !ok {
		return fmt.Errorf("Governance transfer %s not found", ID)
	}
	e.deleteGovTransfer(ID)
	gTransfer.Status = types.TransferStatusCancelled
	e.broker.Send(events.NewGovTransferFundsEvent(ctx, gTransfer, num.UintZero()))
	return nil
}
