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
