// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package subscribers

import (
	"context"

	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
)

type TransferStore interface {
	AddTransfer(eventspb.Transfer)
}

type TransferEvent interface {
	events.Event
	Proto() eventspb.Transfer
}

type TransferSub struct {
	*Base

	store TransferStore

	log *logging.Logger
}

func NewTransferSub(
	ctx context.Context,
	store TransferStore,
	log *logging.Logger,
	ack bool,
) *TransferSub {
	sub := &TransferSub{
		Base:  NewBase(ctx, 10, ack),
		store: store,
		log:   log,
	}

	if sub.isRunning() {
		go sub.loop(ctx)
	}

	return sub
}

func (t *TransferSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			t.Halt()
			return
		case e := <-t.ch:
			if t.isRunning() {
				t.Push(e...)
			}
		}
	}
}

func (t *TransferSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}

	for _, e := range evts {
		switch et := e.(type) {
		case TransferEvent:
			t.store.AddTransfer(et.Proto())
		default:
			t.log.Panic("Unknown event type in transfers subscriber", logging.String("Type", et.Type().String()))
		}
	}
}

func (db *TransferSub) Types() []events.Type {
	return []events.Type{
		events.TransferEvent,
	}
}
