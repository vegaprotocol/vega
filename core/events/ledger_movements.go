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

func (t LedgerMovements) IsParty(id string) bool {
	for _, r := range t.ledgerMovements {
		for _, e := range r.Entries {
			if *e.FromAccount.Owner == id || *e.ToAccount.Owner == id {
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

func (t LedgerMovements) StreamMessage() *eventspb.BusEvent {
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
