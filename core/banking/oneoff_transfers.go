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
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
)

var ErrUnsupportedTransferKind = errors.New("unsupported transfer kind")

type scheduledTransferInstruction struct {
	// to send events
	oneoff              *types.OneOffTransferInstruction
	transferInstruction *types.TransferInstruction
	accountType         types.AccountType
	reference           string
}

func (s *scheduledTransferInstruction) ToProto() *checkpoint.ScheduledTransferInstruction {
	return &checkpoint.ScheduledTransferInstruction{
		OneoffTransferInstruction: s.oneoff.IntoEvent(),
		TransferInstruction:       s.transferInstruction.IntoProto(),
		AccountType:               s.accountType,
		Reference:                 s.reference,
	}
}

func scheduledTransferInstructionFromProto(p *checkpoint.ScheduledTransferInstruction) (scheduledTransferInstruction, error) {
	transferInstruction, err := types.TransferFromProto(p.TransferInstruction)
	if err != nil {
		return scheduledTransferInstruction{}, err
	}

	return scheduledTransferInstruction{
		oneoff:              types.OneOffTransferInstructionFromEvent(p.OneoffTransferInstruction),
		transferInstruction: transferInstruction,
		accountType:         p.AccountType,
		reference:           p.Reference,
	}, nil
}

func (e *Engine) oneOffTransferInstruction(
	ctx context.Context,
	transferInstruction *types.OneOffTransferInstruction,
) error {
	defer func() {
		e.broker.Send(events.NewOneOffTransferInstructionFundsEvent(ctx, transferInstruction))
	}()

	// ensure asset exists
	a, err := e.assets.Get(transferInstruction.Asset)
	if err != nil {
		transferInstruction.Status = types.TransferInstructionStatusRejected
		e.log.Debug("cannot transfer funds, invalid asset", logging.Error(err))
		return fmt.Errorf("could not transfer funds: %w", err)
	}

	if err := transferInstruction.IsValid(); err != nil {
		transferInstruction.Status = types.TransferInstructionStatusRejected
		return err
	}

	if err := e.ensureMinimalTransferAmount(a, transferInstruction.Amount); err != nil {
		transferInstruction.Status = types.TransferInstructionStatusRejected
		return err
	}

	tresps, err := e.processTransfer(
		ctx, transferInstruction.From, transferInstruction.To, transferInstruction.Asset, "", transferInstruction.FromAccountType,
		transferInstruction.ToAccountType, transferInstruction.Amount, transferInstruction.Reference, transferInstruction,
	)
	if err != nil {
		transferInstruction.Status = types.TransferInstructionStatusRejected
		return err
	}

	// all was OK
	transferInstruction.Status = types.TransferInstructionStatusDone
	e.broker.Send(events.NewTransferInstructionResponse(ctx, tresps))

	return nil
}

type timesToTransferInstructions struct {
	deliverOn time.Time
	transfer  []scheduledTransferInstruction
}

func (e *Engine) distributeScheduledTransferInstructions(ctx context.Context) error {
	ttfs := []timesToTransferInstructions{}

	// iterate over those scheduled transfers to sort them by time
	now := e.timeService.GetTimeNow()
	for k, v := range e.scheduledTransferInstructions {
		if !now.Before(k) {
			ttfs = append(ttfs, timesToTransferInstructions{k, v})
			delete(e.scheduledTransferInstructions, k)
		}
	}

	// sort slice by time.
	// no need to sort transfers they are going out as first in first out.
	sort.SliceStable(ttfs, func(i, j int) bool {
		return ttfs[i].deliverOn.Before(ttfs[j].deliverOn)
	})

	transfersInstructions := []*types.TransferInstruction{}
	accountTypes := []types.AccountType{}
	references := []string{}
	evts := []events.Event{}
	for _, v := range ttfs {
		for _, t := range v.transfer {
			t.oneoff.Status = types.TransferInstructionStatusDone
			evts = append(evts, events.NewOneOffTransferInstructionFundsEvent(ctx, t.oneoff))
			transfersInstructions = append(transfersInstructions, t.transfer)
			accountTypes = append(accountTypes, t.accountType)
			references = append(references, t.reference)
		}
	}

	if len(transfersInstructions) <= 0 {
		// nothing to do yeay
		return nil
	}

	// at least 1 transfer updated, set to true
	e.bss.changedScheduledTransferInstructions = true
	tresps, err := e.col.TransferFunds(
		ctx, transfersInstructions, accountTypes, references, nil, nil, // no fees required there, they've been paid already
	)
	if err != nil {
		return err
	}

	e.broker.Send(events.NewTransferInstructionResponse(ctx, tresps))
	e.broker.SendBatch(evts)

	return nil
}

func (e *Engine) scheduleTransferInstruction(
	oneoff *types.OneOffTransferInstruction,
	t *types.TransferInstruction,
	ty types.AccountType,
	reference string,
	deliverOn time.Time,
) {
	sts, ok := e.scheduledTransferInstructions[deliverOn]
	if !ok {
		e.scheduledTransferInstructions[deliverOn] = []scheduledTransferInstruction{}
		sts = e.scheduledTransferInstructions[deliverOn]
	}

	sts = append(sts, scheduledTransferInstruction{
		oneoff:      oneoff,
		transfer:    t,
		accountType: ty,
		reference:   reference,
	})
	e.scheduledTransferInstructions[deliverOn] = sts
	e.bss.changedScheduledTransferInstructions = true
}
