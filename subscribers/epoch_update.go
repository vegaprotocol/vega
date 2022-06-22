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
	types "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
)

type EpochUpdateEvent interface {
	events.Event
	Proto() eventspb.EpochEvent
}

type EpochStore interface {
	AddEpoch(seq uint64, startTime int64, expiryTime int64, endTime int64)
	AddDelegation(types.Delegation)
}

type EpochUpdateSub struct {
	*Base

	epochStore EpochStore

	log *logging.Logger
}

func NewEpochUpdateSub(ctx context.Context, epochStore EpochStore, log *logging.Logger, ack bool) *EpochUpdateSub {
	sub := &EpochUpdateSub{
		Base:       NewBase(ctx, 10, ack),
		epochStore: epochStore,
		log:        log,
	}

	if sub.isRunning() {
		go sub.loop(ctx)
	}

	return sub
}

func (vu *EpochUpdateSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			vu.Halt()
			return
		case e := <-vu.ch:
			if vu.isRunning() {
				vu.Push(e...)
			}
		}
	}
}

func (vu *EpochUpdateSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}

	for _, e := range evts {
		switch et := e.(type) {
		case EpochUpdateEvent:
			eu := et.Proto()
			vu.epochStore.AddEpoch(eu.GetSeq(), eu.GetStartTime(), eu.GetExpireTime(), eu.GetEndTime())
		default:
			vu.log.Panic("Unknown event type in epoch event subscriber", logging.String("Type", et.Type().String()))
		}
	}
}

func (vu *EpochUpdateSub) Types() []events.Type {
	return []events.Type{
		events.EpochUpdate,
	}
}
