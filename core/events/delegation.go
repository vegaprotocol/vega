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

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"
)

type DelegationBalance struct {
	*Base
	Party    string
	NodeID   string
	Amount   *num.Uint
	EpochSeq string
}

func NewDelegationBalance(ctx context.Context, party, nodeID string, amount *num.Uint, epochSeq string) *DelegationBalance {
	return &DelegationBalance{
		Base:     newBase(ctx, DelegationBalanceEvent),
		Party:    party,
		NodeID:   nodeID,
		Amount:   amount,
		EpochSeq: epochSeq,
	}
}

func (db DelegationBalance) Proto() eventspb.DelegationBalanceEvent {
	return eventspb.DelegationBalanceEvent{
		Party:    db.Party,
		NodeId:   db.NodeID,
		Amount:   db.Amount.String(),
		EpochSeq: db.EpochSeq,
	}
}

func (db DelegationBalance) StreamMessage() *eventspb.BusEvent {
	p := db.Proto()
	busEvent := newBusEventFromBase(db.Base)
	busEvent.Event = &eventspb.BusEvent_DelegationBalance{
		DelegationBalance: &p,
	}
	return busEvent
}

func DelegationBalanceEventFromStream(ctx context.Context, be *eventspb.BusEvent) *DelegationBalance {
	event := be.GetDelegationBalance()
	if event == nil {
		return nil
	}

	amt, err := num.UintFromString(event.GetAmount(), 10)
	if err {
		return nil
	}

	return &DelegationBalance{
		Base:     newBaseFromBusEvent(ctx, DelegationBalanceEvent, be),
		Party:    event.GetParty(),
		NodeID:   event.GetNodeId(),
		Amount:   amt,
		EpochSeq: event.GetEpochSeq(),
	}
}
