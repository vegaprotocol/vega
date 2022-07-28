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

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/libs/proto"
	"code.vegaprotocol.io/vega/core/types"
)

func (e *Engine) Name() types.CheckpointName {
	return types.BankingCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	msg := &checkpoint.Banking{
		TransfersAtTime:    e.getScheduledTransfers(),
		RecurringTransfers: e.getRecurringTransfers(),
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

	evts, err := e.loadScheduledTransfers(ctx, b.TransfersAtTime)
	if err != nil {
		return err
	}

	evts = append(evts, e.loadRecurringTransfers(ctx, b.RecurringTransfers)...)

	if len(evts) > 0 {
		e.broker.SendBatch(evts)
	}

	return nil
}

func (e *Engine) loadScheduledTransfers(
	ctx context.Context, r []*checkpoint.ScheduledTransferAtTime,
) ([]events.Event, error) {
	evts := []events.Event{}
	for _, v := range r {
		transfers := make([]scheduledTransfer, 0, len(v.Transfers))
		for _, v := range v.Transfers {
			transfer, err := scheduledTransferFromProto(v)
			if err != nil {
				return nil, err
			}
			evts = append(evts, events.NewOneOffTransferFundsEvent(ctx, transfer.oneoff))
			transfers = append(transfers, transfer)
		}
		e.scheduledTransfers[time.Unix(v.DeliverOn, 0)] = transfers
	}

	return evts, nil
}

func (e *Engine) loadRecurringTransfers(
	ctx context.Context, r *checkpoint.RecurringTransfers,
) []events.Event {
	evts := []events.Event{}
	for _, v := range r.RecurringTransfers {
		transfer := types.RecurringTransferFromEvent(v)
		e.recurringTransfers[transfer.ID] = transfer
		evts = append(evts, events.NewRecurringTransferFundsEvent(ctx, transfer))
	}
	return evts
}

func (e *Engine) getRecurringTransfers() *checkpoint.RecurringTransfers {
	out := &checkpoint.RecurringTransfers{
		RecurringTransfers: make([]*eventspb.Transfer, 0, len(e.recurringTransfers)),
	}

	for _, v := range e.recurringTransfers {
		out.RecurringTransfers = append(out.RecurringTransfers, v.IntoEvent())
	}

	sort.SliceStable(out.RecurringTransfers, func(i, j int) bool {
		return out.RecurringTransfers[i].Id < out.RecurringTransfers[j].Id
	})

	return out
}

func (e *Engine) getScheduledTransfers() []*checkpoint.ScheduledTransferAtTime {
	out := []*checkpoint.ScheduledTransferAtTime{}

	for k, v := range e.scheduledTransfers {
		transfers := make([]*checkpoint.ScheduledTransfer, 0, len(v))
		for _, v := range v {
			transfers = append(transfers, v.ToProto())
		}

		out = append(out, &checkpoint.ScheduledTransferAtTime{DeliverOn: k.Unix(), Transfers: transfers})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].DeliverOn < out[j].DeliverOn })

	return out
}
