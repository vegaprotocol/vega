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
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
)

var ErrUnsupportedTransferKind = errors.New("unsupported transfer kind")

type scheduledTransfer struct {
	// to send events
	oneoff      *types.OneOffTransfer
	transfer    *types.Transfer
	accountType types.AccountType
	reference   string
}

func (s *scheduledTransfer) ToProto() *checkpoint.ScheduledTransfer {
	return &checkpoint.ScheduledTransfer{
		OneoffTransfer: s.oneoff.IntoEvent(nil),
		Transfer:       s.transfer.IntoProto(),
		AccountType:    s.accountType,
		Reference:      s.reference,
	}
}

func scheduledTransferFromProto(p *checkpoint.ScheduledTransfer) (scheduledTransfer, error) {
	transfer, err := types.TransferFromProto(p.Transfer)
	if err != nil {
		return scheduledTransfer{}, err
	}

	return scheduledTransfer{
		oneoff:      types.OneOffTransferFromEvent(p.OneoffTransfer),
		transfer:    transfer,
		accountType: p.AccountType,
		reference:   p.Reference,
	}, nil
}

func (e *Engine) oneOffTransfer(
	ctx context.Context,
	transfer *types.OneOffTransfer,
) (err error) {
	defer func() {
		if err != nil {
			e.broker.Send(events.NewOneOffTransferFundsEventWithReason(ctx, transfer, err.Error()))
		} else {
			e.broker.Send(events.NewOneOffTransferFundsEvent(ctx, transfer))
		}
	}()

	// ensure asset exists
	a, err := e.assets.Get(transfer.Asset)
	if err != nil {
		transfer.Status = types.TransferStatusRejected
		e.log.Debug("cannot transfer funds, invalid asset", logging.Error(err))
		return fmt.Errorf("could not transfer funds: %w", err)
	}

	if err := transfer.IsValid(); err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	if err := e.ensureMinimalTransferAmount(a, transfer.Amount, transfer.FromAccountType, transfer.From); err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	tresps, err := e.processTransfer(
		ctx, a, transfer.From, transfer.To, "", transfer.FromAccountType,
		transfer.ToAccountType, transfer.Amount, transfer.Reference, transfer.ID, e.currentEpoch, transfer,
	)
	if err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	// all was OK
	transfer.Status = types.TransferStatusDone
	e.broker.Send(events.NewLedgerMovements(ctx, tresps))

	return nil
}

type timesToTransfers struct {
	deliverOn int64
	transfer  []scheduledTransfer
}

func (e *Engine) distributeScheduledTransfers(ctx context.Context, now time.Time) error {
	ttfs := []timesToTransfers{}

	// iterate over those scheduled transfers to sort them by time
	for k, v := range e.scheduledTransfers {
		if now.UnixNano() >= k {
			ttfs = append(ttfs, timesToTransfers{k, v})
			delete(e.scheduledTransfers, k)
		}
	}

	// sort slice by time.
	// no need to sort transfers they are going out as first in first out.
	sort.SliceStable(ttfs, func(i, j int) bool {
		return ttfs[i].deliverOn < ttfs[j].deliverOn
	})

	transfers := []*types.Transfer{}
	accountTypes := []types.AccountType{}
	references := []string{}
	evts := []events.Event{}
	for _, v := range ttfs {
		for _, t := range v.transfer {
			t.oneoff.Status = types.TransferStatusDone
			evts = append(evts, events.NewOneOffTransferFundsEvent(ctx, t.oneoff))
			transfers = append(transfers, t.transfer)
			accountTypes = append(accountTypes, t.accountType)
			references = append(references, t.reference)
		}
	}

	if len(transfers) <= 0 {
		// nothing to do yeay
		return nil
	}

	// at least 1 transfer updated, set to true
	tresps, err := e.col.TransferFunds(
		ctx, transfers, accountTypes, references, nil, nil, // no fees required there, they've been paid already
	)
	if err != nil {
		return err
	}

	e.broker.Send(events.NewLedgerMovements(ctx, tresps))
	e.broker.SendBatch(evts)

	return nil
}

func (e *Engine) scheduleTransfer(
	oneoff *types.OneOffTransfer,
	t *types.Transfer,
	ty types.AccountType,
	reference string,
	deliverOn time.Time,
) {
	sts, ok := e.scheduledTransfers[deliverOn.UnixNano()]
	if !ok {
		e.scheduledTransfers[deliverOn.UnixNano()] = []scheduledTransfer{}
		sts = e.scheduledTransfers[deliverOn.UnixNano()]
	}

	sts = append(sts, scheduledTransfer{
		oneoff:      oneoff,
		transfer:    t,
		accountType: ty,
		reference:   reference,
	})
	e.scheduledTransfers[deliverOn.UnixNano()] = sts
}
