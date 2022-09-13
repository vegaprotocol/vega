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
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

func (e *Engine) Name() types.CheckpointName {
	return types.BankingCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	msg := &checkpoint.Banking{
		TransferInstructionsAtTime:    e.getScheduledTransferInstructions(),
		RecurringTransferInstructions: e.getRecurringTransferInstructions(),
		BridgeState:                   e.getBridgeState(),
	}
	ret, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	b := checkpoint.Banking{}
	if err := proto.Unmarshal(data, &b); err != nil {
		return err
	}

	evts, err := e.loadScheduledTransferInstructions(ctx, b.TransferInstructionsAtTime)
	if err != nil {
		return err
	}

	evts = append(evts, e.loadRecurringTransferInstructions(ctx, b.RecurringTransferInstructions)...)

	e.loadBridgeState(b.BridgeState)

	if len(evts) > 0 {
		e.broker.SendBatch(evts)
	}

	return nil
}

func (e *Engine) loadBridgeState(state *checkpoint.BridgeState) {
	// this would eventually be nil if we restore from a checkpoint
	// which have been produce from an old version of the core.
	// we set it to active by default in the case
	if state == nil {
		e.bridgeState = &bridgeState{
			active: true,
		}
		return
	}

	e.bridgeState = &bridgeState{
		active:   state.Active,
		block:    state.BlockHeight,
		logIndex: state.LogIndex,
	}
}

func (e *Engine) loadScheduledTransferInstructions(
	ctx context.Context, r []*checkpoint.ScheduledTransferInstructionAtTime,
) ([]events.Event, error) {
	evts := []events.Event{}
	for _, v := range r {
		transferInstructions := make([]scheduledTransferInstruction, 0, len(v.TransferInstructions))
		for _, v := range v.TransferInstructions {
			transferInstruction, err := scheduledTransferInstructionFromProto(v)
			if err != nil {
				return nil, err
			}
			evts = append(evts, events.NewOneOffTransferInstructionFundsEvent(ctx, transferInstruction.oneoff))
			transferInstructions = append(transferInstructions, transferInstruction)
		}
		e.scheduledTransferInstructions[time.Unix(v.DeliverOn, 0)] = transferInstructions
	}

	return evts, nil
}

func (e *Engine) loadRecurringTransferInstructions(
	ctx context.Context, r *checkpoint.RecurringTransferInstructions,
) []events.Event {
	evts := []events.Event{}
	for _, v := range r.RecurringTransferInstructions {
		transferInstruction := types.RecurringTransferInstructionFromEvent(v)
		e.recurringTransferInstructions = append(e.recurringTransferInstructions, transferInstruction)
		e.recurringTransferInstructionsMap[transferInstruction.ID] = transferInstruction
		evts = append(evts, events.NewRecurringTransferFundsEvent(ctx, transferInstruction))
	}
	return evts
}

func (e *Engine) getBridgeState() *checkpoint.BridgeState {
	return &checkpoint.BridgeState{
		Active:      e.bridgeState.active,
		BlockHeight: e.bridgeState.block,
		LogIndex:    e.bridgeState.logIndex,
	}
}

func (e *Engine) getRecurringTransferInstructions() *checkpoint.RecurringTransferInstructions {
	out := &checkpoint.RecurringTransferInstructions{
		RecurringTransferInstructions: make([]*eventspb.TransferInstruction, 0, len(e.recurringTransferInstructions)),
	}

	for _, v := range e.recurringTransferInstructions {
		out.RecurringTransferInstructions = append(out.RecurringTransferInstructions, v.IntoEvent())
	}

	return out
}

func (e *Engine) getScheduledTransferInstructions() []*checkpoint.ScheduledTransferInstructionAtTime {
	out := []*checkpoint.ScheduledTransferInstructionAtTime{}

	for k, v := range e.scheduledTransferInstructions {
		transferInstructions := make([]*checkpoint.ScheduledTransferInstruction, 0, len(v))
		for _, v := range v {
			transferInstructions = append(transferInstructions, v.ToProto())
		}

		out = append(out, &checkpoint.ScheduledTransferInstructionAtTime{DeliverOn: k.Unix(), TransferInstructions: transferInstructions})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].DeliverOn < out[j].DeliverOn })

	return out
}
