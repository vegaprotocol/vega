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

type DelegationStore interface {
	AddDelegation(types.Delegation)
}

type DelegationBalanceEvent interface {
	events.Event
	Proto() eventspb.DelegationBalanceEvent
}

type DelegationBalanceSub struct {
	*Base

	epochStore      EpochStore
	nodeStore       NodeStore
	delegationStore DelegationStore

	log *logging.Logger
}

func NewDelegationBalanceSub(
	ctx context.Context,
	nodeStore NodeStore,
	epochStore EpochStore,
	delegationStore DelegationStore,
	log *logging.Logger,
	ack bool,
) *DelegationBalanceSub {
	sub := &DelegationBalanceSub{
		Base:            NewBase(ctx, 10, ack),
		nodeStore:       nodeStore,
		epochStore:      epochStore,
		delegationStore: delegationStore,
		log:             log,
	}

	if sub.isRunning() {
		go sub.loop(ctx)
	}

	return sub
}

func (db *DelegationBalanceSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			db.Halt()
			return
		case e := <-db.ch:
			if db.isRunning() {
				db.Push(e...)
			}
		}
	}
}

func (db *DelegationBalanceSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}

	for _, e := range evts {
		switch et := e.(type) {
		case DelegationBalanceEvent:
			dbe := et.Proto()

			delegation := types.Delegation{
				EpochSeq: dbe.GetEpochSeq(),
				Party:    dbe.GetParty(),
				NodeId:   dbe.GetNodeId(),
				Amount:   dbe.GetAmount(),
			}

			db.nodeStore.AddDelegation(delegation)
			db.epochStore.AddDelegation(delegation)
			db.delegationStore.AddDelegation(delegation)
		default:
			db.log.Panic("Unknown event type in candles subscriber", logging.String("Type", et.Type().String()))
		}
	}
}

func (db *DelegationBalanceSub) Types() []events.Type {
	return []events.Type{
		events.DelegationBalanceEvent,
	}
}
