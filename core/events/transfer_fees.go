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

// TransferFees ...
type TransferFees struct {
	*Base
	pb *eventspb.TransferFees
}

func NewTransferFeesEvent(ctx context.Context, transferID, asset, partyID string, amount *num.Uint) *TransferFees {
	return &TransferFees{
		Base: newBase(ctx, TransferFeesEvent),
		pb: &eventspb.TransferFees{
			TransferId: transferID,
			Asset:      asset,
			Amount:     amount.String(),
			PartyId:    partyID,
		},
	}
}

func (t TransferFees) Proto() eventspb.TransferFees {
	return *t.pb
}

func (t TransferFees) StreamMessage() *eventspb.BusEvent {
	p := t.Proto()

	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_TransferFees{
		TransferFees: t.pb,
	}

	return busEvent
}

func TransferFeesEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TransferFees {
	event := be.GetTransferFees()
	if event == nil {
		return nil
	}

	return &TransferFees{
		Base: newBaseFromBusEvent(ctx, TransferFeesEvent, be),
		pb:   event,
	}
}
