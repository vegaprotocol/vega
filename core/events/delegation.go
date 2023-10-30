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

package events

import (
	"context"

	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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
