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

	"code.vegaprotocol.io/vega/core/types"
	ptypes "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type LedgerMovements struct {
	*Base
	ledgerMovements []*ptypes.LedgerMovement
}

// NewLedgerMovements returns an event with transfer responses - this is the replacement of the transfer buffer.
func NewLedgerMovements(ctx context.Context, ledgerMovements []*types.LedgerMovement) *LedgerMovements {
	return &LedgerMovements{
		Base:            newBase(ctx, LedgerMovementsEvent),
		ledgerMovements: types.LedgerMovements(ledgerMovements).IntoProto(),
	}
}

// LedgerMovements returns the actual event payload.
func (t *LedgerMovements) LedgerMovements() []*ptypes.LedgerMovement {
	return t.ledgerMovements
}

func (t *LedgerMovements) IsParty(id string) bool {
	isParty := func(owner *string) bool { return owner != nil && *owner == id }

	for _, r := range t.ledgerMovements {
		for _, e := range r.Entries {
			if isParty(e.FromAccount.Owner) || isParty(e.ToAccount.Owner) {
				return true
			}
		}
	}
	return false
}

func (t *LedgerMovements) Proto() eventspb.LedgerMovements {
	return eventspb.LedgerMovements{
		LedgerMovements: t.ledgerMovements,
	}
}

func (t *LedgerMovements) StreamMessage() *eventspb.BusEvent {
	p := t.Proto()
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_LedgerMovements{
		LedgerMovements: &p,
	}

	return busEvent
}

func TransferResponseEventFromStream(ctx context.Context, be *eventspb.BusEvent) *LedgerMovements {
	return &LedgerMovements{
		Base:            newBaseFromBusEvent(ctx, LedgerMovementsEvent, be),
		ledgerMovements: be.GetLedgerMovements().LedgerMovements,
	}
}
